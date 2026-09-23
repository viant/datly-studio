package reader

import (
	"context"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	requestprovider "github.com/viant/bindly/provider/request"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly/bootstrap"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
	dtag "github.com/viant/datly/tag"
)

func TestResourcePolicyReaderGeneratedShapes(t *testing.T) {
	input := reflect.TypeOf(Input{})
	for _, name := range []string{"TenantId", "ResourceKind", "ResourceId", "ResourceVersion"} {
		field, ok := input.FieldByName(name)
		if !ok || !strings.Contains(field.Tag.Get("parameter"), "required=true") || !strings.HasPrefix(field.Tag.Get("predicate"), "equal,h,") {
			t.Fatalf("reader key %s must be a required equality predicate: %q", name, field.Tag)
		}
	}
	if _, ok := input.FieldByName("Jwt"); ok {
		t.Fatal("server-owned policy reader must not accept a client credential")
	}
	view := reflect.TypeOf(ResourcePolicy{})
	for name, want := range map[string]reflect.Type{"TenantId": reflect.TypeOf(""), "ResourceKind": reflect.TypeOf(""), "ResourceId": reflect.TypeOf(""), "ResourceVersion": reflect.TypeOf(""), "ActorId": reflect.TypeOf(""), "Revision": reflect.TypeOf((*int)(nil))} {
		field, ok := view.FieldByName(name)
		if !ok || field.Type != want {
			t.Fatalf("view %s type=%v want %v", name, field.Type, want)
		}
	}
	policies, _ := view.FieldByName("PoliciesJson")
	if !strings.Contains(policies.Tag.Get("sqlx"), "enc=JSON") {
		t.Fatalf("policies must decode as JSON: %q", policies.Tag)
	}
}

func TestResourcePolicyReaderReadsExactIdentityOnly(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "reader", "studio")
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "resource_policy_heads", Rows: []datatest.Row{
			{"tenant_id": "one", "resource_kind": "component", "resource_id": "shared", "resource_version": "1", "revision": 2},
			{"tenant_id": "one", "resource_kind": "skill", "resource_id": "shared", "resource_version": "1", "revision": 1},
			{"tenant_id": "two", "resource_kind": "component", "resource_id": "shared", "resource_version": "1", "revision": 1},
		}},
		datatest.Table{Name: "resource_policy_revisions", Rows: []datatest.Row{
			{"tenant_id": "one", "resource_kind": "component", "resource_id": "shared", "resource_version": "1", "revision": 1, "policies_json": `{"execute":{"mode":"public"}}`, "actor_id": "bootstrap", "occurred_at": "2026-09-23 10:00:00"},
			{"tenant_id": "one", "resource_kind": "component", "resource_id": "shared", "resource_version": "1", "revision": 2, "policies_json": `{"execute":{"mode":"protected"}}`, "actor_id": "publisher", "occurred_at": "2026-09-23 11:00:00"},
			{"tenant_id": "one", "resource_kind": "skill", "resource_id": "shared", "resource_version": "1", "revision": 1, "policies_json": `{"retrieve":{"mode":"public"}}`, "actor_id": "bootstrap", "occurred_at": "2026-09-23 10:00:00"},
			{"tenant_id": "two", "resource_kind": "component", "resource_id": "shared", "resource_version": "1", "revision": 1, "policies_json": `{"execute":{"mode":"public"}}`, "actor_id": "other", "occurred_at": "2026-09-23 10:00:00"},
		}},
	); err != nil {
		t.Fatal(err)
	}
	holder := reflect.TypeOf(PolicyComponent{})
	field, _ := holder.FieldByName("Contract")
	metadata, _, err := dtag.ParseComponent(field.Tag)
	if err != nil {
		t.Fatal(err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackageName: "reader", PackagePath: holder.PkgPath(), Tag: metadata, InputType: "Input", OutputType: "Output"}).Resolve(reflect.TypeOf(Input{}), reflect.TypeOf(Output{}))
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(PolicyDatlyResourceNamespace, PolicyDatlyResources); err != nil {
		t.Fatal(err)
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeOf(Input{}), OutputType: reflect.TypeOf(Output{}), Resources: resources})
	if err != nil {
		t.Fatal(err)
	}
	execution, err := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: &dsql.SQLComponent{DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	registered, err := artifact.Registration(registry.RegisteredComponent{Reader: execution})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registered}, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })
	read := func(values url.Values) (*Output, error) {
		request := httptest.NewRequest("GET", "/v1/studio/resource-policies?"+values.Encode(), nil)
		scope, scopeErr := requestprovider.New(request)
		if scopeErr != nil {
			t.Fatal(scopeErr)
		}
		defer scope.Close()
		actual, readErr := runtime.ExecuteRoute(ctx, "GET", "/v1/studio/resource-policies", scope)
		if readErr != nil {
			return nil, readErr
		}
		return actual.(*Output), nil
	}
	exact := url.Values{"tenantId": {"one"}, "resourceKind": {"component"}, "resourceId": {"shared"}, "resourceVersion": {"1"}}
	output, err := read(exact)
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Policies) != 1 || output.Policies[0].Revision == nil || *output.Policies[0].Revision != 2 || output.Policies[0].ActorId != "publisher" || !strings.Contains(string(output.Policies[0].PoliciesJson), "protected") {
		t.Fatalf("policies=%+v", output.Policies)
	}
	for name, values := range map[string]url.Values{
		"other kind":    {"tenantId": {"one"}, "resourceKind": {"report"}, "resourceId": {"shared"}, "resourceVersion": {"1"}},
		"other tenant":  {"tenantId": {"three"}, "resourceKind": {"component"}, "resourceId": {"shared"}, "resourceVersion": {"1"}},
		"other version": {"tenantId": {"one"}, "resourceKind": {"component"}, "resourceId": {"shared"}, "resourceVersion": {"2"}},
	} {
		t.Run(name, func(t *testing.T) {
			output, err := read(values)
			if err != nil {
				t.Fatal(err)
			}
			if len(output.Policies) != 0 {
				t.Fatalf("resource boundary crossed: %+v", output.Policies)
			}
		})
	}
	for _, missing := range []string{"tenantId", "resourceKind", "resourceId", "resourceVersion"} {
		t.Run("missing "+missing, func(t *testing.T) {
			partial := url.Values{}
			for key, value := range exact {
				if key != missing {
					partial[key] = value
				}
			}
			if output, err := read(partial); err == nil {
				t.Fatalf("missing %s widened the read to %d heads", missing, len(output.Policies))
			}
		})
	}
}
