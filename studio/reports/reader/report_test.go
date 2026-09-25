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
	"github.com/viant/datly/mcp"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
	dtag "github.com/viant/datly/tag"
)

func TestReportReaderMinimumContract(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "reader", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{
			{"name": "main", "driver": "sqlite", "owner_id": "owner-a", "status": "active", "etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"},
			{"name": "archive", "driver": "mysql", "owner_id": "owner-b", "status": "disabled", "etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"},
		}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{
			{"owner_id": "owner-a", "name": "general", "title": "General", "status": "active", "etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"},
			{"owner_id": "owner-b", "name": "general", "title": "General", "status": "active", "etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"},
		}},
		datatest.Table{Name: "reports", Rows: []datatest.Row{
			{"id": "r-alpha", "slug": "alpha", "title": "Alpha analytics", "description": "Primary", "owner_id": "owner-a", "status": "active", "default_connector_name": "main", "component_scope": "reports/alpha", "component_name": "alpha", "current_draft_version": 2, "etag": 1, "created_at": "2026-09-17 10:00:00", "updated_at": "2026-09-17 10:00:00"},
			{"id": "r-beta", "slug": "beta", "title": "Beta archive", "description": "Historical analytics", "owner_id": "owner-b", "status": "disabled", "default_connector_name": "archive", "component_scope": "reports/beta", "component_name": "beta", "etag": 2, "created_at": "2026-09-17 11:00:00", "updated_at": "2026-09-17 11:00:00"},
			{"id": "r-gamma", "slug": "gamma", "title": "Gamma analytics", "owner_id": "owner-a", "status": "draft", "default_connector_name": "main", "component_scope": "reports/gamma", "component_name": "gamma", "current_draft_version": 1, "etag": 3, "created_at": "2026-09-17 12:00:00", "updated_at": "2026-09-17 12:00:00"},
			{"id": "r-removed", "slug": "removed", "title": "Removed", "owner_id": "owner-a", "status": "archived", "default_connector_name": "main", "component_scope": "reports/removed", "component_name": "removed", "etag": 4, "created_at": "2026-09-17 13:00:00", "updated_at": "2026-09-17 13:00:00", "deleted_at": "2026-09-17 13:30:00"},
		}},
		datatest.Table{Name: "report_acl", Rows: []datatest.Row{
			{"report_id": "r-alpha", "subject_type": "user", "subject_id": "viewer", "can_view": true},
			{"report_id": "r-beta", "subject_type": "user", "subject_id": "viewer", "can_view": true},
			{"report_id": "r-gamma", "subject_type": "user", "subject_id": "viewer", "can_view": true},
		}},
	); err != nil {
		t.Fatal(err)
	}

	holder := reflect.TypeOf(ReportComponent{})
	field, ok := holder.FieldByName("Contract1")
	if !ok {
		t.Fatal("missing report reader component holder")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatal(err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackageName: "reader", PackagePath: holder.PkgPath(), Tag: metadata, InputType: "Input", OutputType: "Output"}).Resolve(reflect.TypeOf(Input{}), reflect.TypeOf(Output{}))
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(ReportDatlyResourceNamespace, ReportDatlyResources); err != nil {
		t.Fatal(err)
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeOf(Input{}), OutputType: reflect.TypeOf(Output{}), Resources: resources, Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwt.Factory})
	if err != nil {
		t.Fatal(err)
	}
	execution, err := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: &dsql.SQLComponent{DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	entry, err := artifact.Registration(registry.RegisteredComponent{Reader: execution})
	if err != nil {
		t.Fatal(err)
	}
	authEntry := datatest.AuthRegistration(t, db, jwt.Factory, resources)
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{authEntry, entry}, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })
	mcpService, err := mcp.New(mcp.Config{Components: []*registry.RegisteredComponent{authEntry, entry}, Invoker: runtime, Resources: resources})
	if err != nil {
		t.Fatal(err)
	}
	if names := mcpService.Catalog().ToolNames(); !reflect.DeepEqual(names, []string{"studio.reports.read", "studio.reports.readById"}) {
		t.Fatalf("public report MCP tools=%v", names)
	}

	invokeAs := func(subject, path string) (*Output, error) {
		request := httptest.NewRequest("GET", path, nil)
		request.Header.Set("Authorization", jwt.Bearer(t, subject))
		routePath := "/v1/studio/reports"
		var pathParams map[string]string
		if request.URL.Path != routePath {
			routePath += "/{id}"
			pathParams = map[string]string{"id": strings.TrimPrefix(request.URL.Path, "/v1/studio/reports/")}
		}
		scope, scopeErr := requestprovider.New(request, requestprovider.WithPathParams(pathParams))
		if scopeErr != nil {
			t.Fatal(scopeErr)
		}
		defer scope.Close()
		actual, invokeErr := runtime.ExecuteRoute(ctx, "GET", routePath, scope)
		if invokeErr != nil {
			return nil, invokeErr
		}
		return actual.(*Output), nil
	}
	invoke := func(path string) (*Output, error) { return invokeAs("viewer", path) }
	slugs := func(output *Output) []string {
		if output == nil {
			return nil
		}
		result := make([]string, 0, len(output.Reports))
		for _, report := range output.Reports {
			if report != nil && report.Slug != nil {
				result = append(result, *report.Slug)
			}
		}
		return result
	}

	for _, test := range []struct {
		name string
		path string
		want []string
	}{
		{"all predicates absent", "/v1/studio/reports?orderBy=slug", []string{"alpha", "beta", "gamma"}},
		{"by id", "/v1/studio/reports/r-beta", []string{"beta"}},
		{"by id missing", "/v1/studio/reports/missing", []string{}},
		{"search OR group", "/v1/studio/reports?q=analytics&orderBy=slug", []string{"alpha", "beta", "gamma"}},
		{"status predicate", "/v1/studio/reports?status=active", []string{"alpha"}},
		{"owner predicate", "/v1/studio/reports?owner=owner-a&orderBy=slug", []string{"alpha", "gamma"}},
		{"connector predicate", "/v1/studio/reports?connector=archive", []string{"beta"}},
		{"combined groups", "/v1/studio/reports?q=analytics&owner=owner-a&status=draft", []string{"gamma"}},
		{"explicit empty activates predicate", "/v1/studio/reports?status=", []string{}},
		{"pagination", "/v1/studio/reports?orderBy=slug&limit=1&offset=1", []string{"beta"}},
		{"descending order", "/v1/studio/reports?orderBy=" + url.QueryEscape("slug DESC"), []string{"gamma", "beta", "alpha"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			output, err := invoke(test.path)
			if err != nil {
				t.Fatal(err)
			}
			if actual := slugs(output); !reflect.DeepEqual(actual, test.want) {
				t.Fatalf("%s slugs=%v want=%v", test.path, actual, test.want)
			}
		})
	}

	t.Run("field projection", func(t *testing.T) {
		output, err := invoke("/v1/studio/reports?fields=slug&fields=status&orderBy=slug&limit=1")
		if err != nil {
			t.Fatal(err)
		}
		if len(output.Reports) != 1 || output.Reports[0].Slug == nil || output.Reports[0].Status == nil || output.Reports[0].Title != nil {
			t.Fatalf("projected report=%+v", output.Reports)
		}
	})

	t.Run("disallowed order", func(t *testing.T) {
		if _, err := invoke("/v1/studio/reports?orderBy=deleted_at"); err == nil {
			t.Fatal("disallowed order column was accepted")
		}
	})

	t.Run("predicate presence markers", func(t *testing.T) {
		input := Input{}
		input.SetStatus("")
		if input.Has == nil || !input.Has.Status || input.Has.Query || input.Has.ConnectorName {
			t.Fatalf("predicate presence=%+v", input.Has)
		}
	})

	t.Run("verified principal scopes public report catalog", func(t *testing.T) {
		if subject, scoped := (Input{}).ReportCatalogScope(); subject != "" || !scoped {
			t.Fatalf("missing trusted auth scope=%q scoped=%v", subject, scoped)
		}
		output, err := invokeAs("owner-a", "/v1/studio/reports?orderBy=slug")
		if err != nil || !reflect.DeepEqual(slugs(output), []string{"alpha", "gamma"}) {
			t.Fatalf("owner-a reports=%v err=%v", slugs(output), err)
		}
		output, err = invokeAs("owner-b", "/v1/studio/reports?orderBy=slug")
		if err != nil || !reflect.DeepEqual(slugs(output), []string{"beta"}) {
			t.Fatalf("owner-b reports=%v err=%v", slugs(output), err)
		}
		if _, err = db.ExecContext(ctx, "DELETE FROM report_acl WHERE report_id = ? AND subject_id = ?", "r-beta", "viewer"); err != nil {
			t.Fatal(err)
		}
		output, err = invoke("/v1/studio/reports?orderBy=slug")
		if err != nil || !reflect.DeepEqual(slugs(output), []string{"alpha", "gamma"}) {
			t.Fatalf("revoked viewer reports=%v err=%v", slugs(output), err)
		}
	})
}
