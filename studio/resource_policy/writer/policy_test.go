package writer

import (
	"context"
	"database/sql"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/viant/bindly/locator"
	requestprovider "github.com/viant/bindly/provider/request"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly/bootstrap"
	druntime "github.com/viant/datly/runtime"
	writerhandler "github.com/viant/datly/runtime/handler/writer"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/sql/dml"
	viewprovider "github.com/viant/datly/sql/reader/provider"
	dtag "github.com/viant/datly/tag"
	"github.com/viant/xdatly/response"
)

func TestResourcePolicyWriterGeneratedShapes(t *testing.T) {
	datatest.AssertFilesAbsent(t, ".", "actions.go", "frames.go", "layout.go", "links.go", "mutation.go", "mutation_output.go", "previous.go", "validation.go", "lifecycle_dispatch.go")
	head := reflect.TypeOf(ResourcePolicyHead{})
	for _, name := range []string{"TenantId", "ResourceKind", "ResourceId", "ResourceVersion"} {
		field, ok := head.FieldByName(name)
		if !ok || !strings.Contains(field.Tag.Get("sqlx"), "primaryKey") || !strings.Contains(field.Tag.Get("validate"), "required") {
			t.Fatalf("head key %s tag=%q", name, field.Tag)
		}
	}
	revision, _ := head.FieldByName("Revision")
	if revision.Tag.Get("writer") != "concurrency" || revision.Type != reflect.TypeOf((*int)(nil)) {
		t.Fatalf("head revision must be the concurrency token: %q %s", revision.Tag, revision.Type)
	}
	history, _ := head.FieldByName("History")
	if !strings.Contains(history.Tag.Get("view"), "table=resource_policy_revisions") || !strings.Contains(history.Tag.Get("on"), "ResourceVersion:head.resource_version=ResourceVersion:history.resource_version") {
		t.Fatalf("history relation tag=%q", history.Tag)
	}
	child := reflect.TypeOf(ResourcePolicyRevision{})
	for _, name := range []string{"TenantId", "ResourceKind", "ResourceId", "ResourceVersion", "Revision"} {
		field, ok := child.FieldByName(name)
		if !ok || !strings.Contains(field.Tag.Get("sqlx"), "primaryKey") {
			t.Fatalf("history key %s tag=%q", name, field.Tag)
		}
	}
	for _, name := range []string{"ActorId", "PoliciesJson", "OccurredAt"} {
		field, _ := child.FieldByName(name)
		if !strings.Contains(field.Tag.Get("validate"), "required") {
			t.Fatalf("history %s must be required: %q", name, field.Tag)
		}
	}
	if _, ok := head.FieldByName("ShouldDelete"); ok {
		t.Fatal("immutable policy history unexpectedly exposes physical deletion")
	}
	input := reflect.TypeOf(Input{})
	for _, name := range []string{"Policies", "PolicyKeys", "CurrentPolicy", "CurrentHistory"} {
		if _, ok := input.FieldByName(name); !ok {
			t.Fatalf("writer input lacks %s", name)
		}
	}
	if _, ok := input.FieldByName("Jwt"); ok {
		t.Fatal("server-owned policy writer must not accept a client credential")
	}
}

func newPolicyWriterRuntime(t *testing.T, db *sql.DB) (*druntime.Runtime, func(string) (*Output, error)) {
	t.Helper()
	ctx := context.Background()
	holder := reflect.TypeOf(PolicyComponent{})
	field, _ := holder.FieldByName("Contract")
	metadata, _, err := dtag.ParseComponent(field.Tag)
	if err != nil {
		t.Fatal(err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackageName: "writer", PackagePath: holder.PkgPath(), Tag: metadata, InputType: "Input", OutputType: "Output"}).Resolve(reflect.TypeOf(Input{}), reflect.TypeOf(Output{}))
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
	sqlComponent := &dsql.SQLComponent{DB: db}
	if err = sqlComponent.RegisterConnector("studio", db); err != nil {
		t.Fatal(err)
	}
	views, err := viewprovider.New(viewprovider.Config{Dependencies: artifact.ViewDependencies, Input: artifact.Input, SQL: sqlComponent})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := writerhandler.New(component, reflect.TypeOf(Input{}), reflect.TypeOf(Output{}), "patch")
	if err != nil {
		t.Fatal(err)
	}
	registered := &registry.RegisteredComponent{Component: artifact.Component, Input: artifact.Input, Output: artifact.Output, OutputType: reflect.TypeOf(Output{}), Handler: handler, Providers: []locator.Provider{views}, DataSource: dml.Source{DB: db}}
	registered.Capabilities.Connector = sqlComponent
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registered}, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })
	invoke := func(body string) (*Output, error) {
		request := httptest.NewRequest("PATCH", "/v1/studio/resource-policies", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		scope, scopeErr := requestprovider.New(request)
		if scopeErr != nil {
			t.Fatal(scopeErr)
		}
		defer scope.Close()
		actual, invokeErr := runtime.ExecuteRoute(ctx, "PATCH", "/v1/studio/resource-policies", scope)
		if invokeErr != nil {
			return nil, invokeErr
		}
		return actual.(*Output), nil
	}
	return runtime, invoke
}

func TestResourcePolicyWriterActivationContract(t *testing.T) {
	ctx := context.Background()
	owner := datatest.OpenSQLite(t, "writer", "studio")
	_, invoke := newPolicyWriterRuntime(t, owner)
	const key = `"tenantId":"one","resourceKind":"component","resourceId":"shared","resourceVersion":"1"`
	history := func(revision int, actor, policies string) string {
		return `{` + key + `,"revision":` + strconv.Itoa(revision) + `,"actorId":"` + actor + `","policiesJson":` + policies + `,"occurredAt":"2026-09-23T10:00:00Z"}`
	}
	status := func(err error) int { return response.ErrorStatusCode(err, 500) }

	t.Run("bootstrap provisions revision one", func(t *testing.T) {
		output, err := invoke(`{"data":[{` + key + `,"revision":0,"history":[` + history(1, "bootstrap", `{"execute":{"mode":"public"}}`) + `]}]}`)
		if err != nil {
			t.Fatal(err)
		}
		if output.Status.Status != "ok" {
			t.Fatalf("output=%+v", output)
		}
		datatest.AssertRows(t, ctx, owner, "SELECT revision FROM resource_policy_heads", nil, datatest.Row{"revision": 1})
		datatest.AssertRows(t, ctx, owner, "SELECT revision,actor_id FROM resource_policy_revisions ORDER BY revision", nil, datatest.Row{"revision": 1, "actor_id": "bootstrap"})
		datatest.AssertRows(t, ctx, owner, "SELECT created_by,updated_by,created_at=updated_at AS same_time FROM resource_policy_heads", nil,
			datatest.Row{"created_by": "bootstrap", "updated_by": "bootstrap", "same_time": 1})
		datatest.AssertRows(t, ctx, owner, "SELECT created_by,updated_by,created_at=occurred_at AS same_time FROM resource_policy_revisions", nil,
			datatest.Row{"created_by": "bootstrap", "updated_by": "bootstrap", "same_time": 1})
	})
	t.Run("bootstrap cannot overwrite", func(t *testing.T) {
		_, err := invoke(`{"data":[{` + key + `,"revision":0,"history":[` + history(1, "bootstrap", `{"execute":{"mode":"public"}}`) + `]}]}`)
		if err == nil || status(err) != 409 {
			t.Fatalf("bootstrap overwrite error=%v", err)
		}
		datatest.AssertRows(t, ctx, owner, "SELECT COUNT(*) AS n FROM resource_policy_revisions", nil, datatest.Row{"n": 1})
	})
	t.Run("replace advances head and appends history atomically", func(t *testing.T) {
		output, err := invoke(`{"data":[{` + key + `,"revision":1,"history":[` + history(2, "publisher", `{"execute":{"mode":"protected","rule":{"kind":"role","value":"reader"}}}`) + `]}]}`)
		if err != nil {
			t.Fatal(err)
		}
		if output.Status.Status != "ok" {
			t.Fatalf("output=%+v", output)
		}
		datatest.AssertRows(t, ctx, owner, "SELECT revision FROM resource_policy_heads", nil, datatest.Row{"revision": 2})
		datatest.AssertRows(t, ctx, owner, "SELECT revision,actor_id FROM resource_policy_revisions ORDER BY revision", nil, datatest.Row{"revision": 1, "actor_id": "bootstrap"}, datatest.Row{"revision": 2, "actor_id": "publisher"})
		datatest.AssertRows(t, ctx, owner, "SELECT created_by,updated_by FROM resource_policy_heads", nil,
			datatest.Row{"created_by": "bootstrap", "updated_by": "publisher"})
		datatest.AssertRows(t, ctx, owner, "SELECT created_by,updated_by,created_at=occurred_at AS same_time FROM resource_policy_revisions WHERE revision=2", nil,
			datatest.Row{"created_by": "publisher", "updated_by": "publisher", "same_time": 1})
	})
	t.Run("stale expected revision conflicts without orphan history", func(t *testing.T) {
		_, err := invoke(`{"data":[{` + key + `,"revision":1,"history":[` + history(2, "stale", `{"execute":{"mode":"public"}}`) + `]}]}`)
		if err == nil || status(err) != 409 {
			t.Fatalf("stale CAS error=%v", err)
		}
		datatest.AssertRows(t, ctx, owner, "SELECT revision FROM resource_policy_heads", nil, datatest.Row{"revision": 2})
		datatest.AssertRows(t, ctx, owner, "SELECT COUNT(*) AS n FROM resource_policy_revisions", nil, datatest.Row{"n": 2})
	})
	t.Run("replace of a missing resource conflicts", func(t *testing.T) {
		_, err := invoke(`{"data":[{"tenantId":"one","resourceKind":"component","resourceId":"absent","resourceVersion":"1","revision":1,"history":[{"tenantId":"one","resourceKind":"component","resourceId":"absent","resourceVersion":"1","revision":2,"actorId":"x","policiesJson":{"execute":{"mode":"public"}},"occurredAt":"2026-09-23T10:00:00Z"}]}]}`)
		if err == nil || status(err) != 409 {
			t.Fatalf("missing head replace error=%v", err)
		}
		datatest.AssertRows(t, ctx, owner, "SELECT COUNT(*) AS n FROM resource_policy_heads", nil, datatest.Row{"n": 1})
	})
	t.Run("activation is denied before mutation when the audit record is invalid", func(t *testing.T) {
		for _, tc := range []struct{ name, body string }{
			{"no history", `{"data":[{` + key + `,"revision":2,"history":[]}]}`},
			{"two history rows", `{"data":[{` + key + `,"revision":2,"history":[` + history(3, "a", `{"x":{}}`) + `,` + history(4, "b", `{"x":{}}`) + `]}]}`},
			{"wrong revision", `{"data":[{` + key + `,"revision":2,"history":[` + history(9, "a", `{"x":{}}`) + `]}]}`},
			{"missing actor", `{"data":[{` + key + `,"revision":2,"history":[` + history(3, "", `{"x":{}}`) + `]}]}`},
			{"array policies", `{"data":[{` + key + `,"revision":2,"history":[` + history(3, "a", `[1,2]`) + `]}]}`},
			{"scalar policies", `{"data":[{` + key + `,"revision":2,"history":[` + history(3, "a", `"public"`) + `]}]}`},
			{"negative expected revision", `{"data":[{` + key + `,"revision":-1,"history":[` + history(0, "a", `{"x":{}}`) + `]}]}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				_, err := invoke(tc.body)
				if err == nil || status(err) < 400 || status(err) >= 500 {
					t.Fatalf("error=%v", err)
				}
				datatest.AssertRows(t, ctx, owner, "SELECT revision FROM resource_policy_heads", nil, datatest.Row{"revision": 2})
				datatest.AssertRows(t, ctx, owner, "SELECT COUNT(*) AS n FROM resource_policy_revisions", nil, datatest.Row{"n": 2})
			})
		}
	})
	t.Run("failed audit insert rolls back the head", func(t *testing.T) {
		if _, err := owner.ExecContext(ctx, `CREATE TRIGGER reject_policy_history BEFORE INSERT ON resource_policy_revisions BEGIN SELECT RAISE(ABORT,'history unavailable'); END`); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _, _ = owner.ExecContext(ctx, `DROP TRIGGER reject_policy_history`) })
		_, err := invoke(`{"data":[{` + key + `,"revision":2,"history":[` + history(3, "publisher", `{"execute":{"mode":"public"}}`) + `]}]}`)
		if err == nil {
			t.Fatal("expected failed audit write")
		}
		datatest.AssertRows(t, ctx, owner, "SELECT revision FROM resource_policy_heads", nil, datatest.Row{"revision": 2})
		datatest.AssertRows(t, ctx, owner, "SELECT COUNT(*) AS n FROM resource_policy_revisions", nil, datatest.Row{"n": 2})
		if _, err := owner.ExecContext(ctx, `DROP TRIGGER reject_policy_history`); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("history keys always belong to the head", func(t *testing.T) {
		// The writer's relation binding owns the child join keys: a history
		// row cannot be attached to a different resource than the head it
		// activates, whatever the caller supplied.
		_, err := invoke(`{"data":[{` + key + `,"revision":2,"createdBy":"mallory","updatedBy":"mallory","history":[{"tenantId":"two","resourceKind":"skill","resourceId":"other","resourceVersion":"9","revision":3,"actorId":"publisher","createdBy":"mallory","updatedBy":"mallory","policiesJson":{"execute":{"mode":"public"}},"occurredAt":"2026-09-23T10:00:00Z"}]}]}`)
		if err != nil {
			t.Fatal(err)
		}
		datatest.AssertRows(t, ctx, owner, "SELECT revision FROM resource_policy_heads", nil, datatest.Row{"revision": 3})
		datatest.AssertRows(t, ctx, owner, "SELECT tenant_id,resource_kind,resource_id,resource_version,revision FROM resource_policy_revisions WHERE revision=3", nil,
			datatest.Row{"tenant_id": "one", "resource_kind": "component", "resource_id": "shared", "resource_version": "1", "revision": 3})
		datatest.AssertRows(t, ctx, owner, "SELECT COUNT(*) AS n FROM resource_policy_revisions WHERE tenant_id<>'one' OR resource_kind<>'component' OR resource_id<>'shared' OR resource_version<>'1'", nil, datatest.Row{"n": 0})
		datatest.AssertRows(t, ctx, owner, "SELECT created_by,updated_by FROM resource_policy_heads", nil,
			datatest.Row{"created_by": "bootstrap", "updated_by": "publisher"})
		datatest.AssertRows(t, ctx, owner, "SELECT created_by,updated_by FROM resource_policy_revisions WHERE revision=3", nil,
			datatest.Row{"created_by": "publisher", "updated_by": "publisher"})
	})
}
