package reader

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	requestprovider "github.com/viant/bindly/provider/request"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly/bootstrap"
	gateway "github.com/viant/datly/gateway/http"
	"github.com/viant/datly/gateway/openapi"
	"github.com/viant/datly/gateway/openapi/openapi3"
	"github.com/viant/datly/mcp"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	dtag "github.com/viant/datly/tag"
	"github.com/viant/mcp-protocol/authorization"
	"github.com/viant/mcp-protocol/schema"
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
	field, ok := holder.FieldByName("Contract")
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
	if names := mcpService.Catalog().ToolNames(); !reflect.DeepEqual(names, []string{"studio.sdk.reports.list"}) {
		t.Fatalf("public report MCP tools=%v", names)
	}

	invokeAs := func(subject string, input map[string]any) (*Output, error) {
		payload, marshalErr := json.Marshal(input)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		request := httptest.NewRequest("POST", "/v1/studio/sdk/reports.list", bytes.NewReader(payload))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", jwt.Bearer(t, subject))
		scope, scopeErr := requestprovider.New(request)
		if scopeErr != nil {
			t.Fatal(scopeErr)
		}
		defer scope.Close()
		actual, invokeErr := runtime.ExecuteRoute(ctx, "POST", "/v1/studio/sdk/reports.list", scope)
		if invokeErr != nil {
			return nil, invokeErr
		}
		return actual.(*Output), nil
	}
	invoke := func(input map[string]any) (*Output, error) { return invokeAs("viewer", input) }
	slugs := func(output *Output) []string {
		if output == nil {
			return nil
		}
		result := make([]string, 0, len(output.Items))
		for _, report := range output.Items {
			if report != nil {
				result = append(result, report.Slug)
			}
		}
		return result
	}

	for _, test := range []struct {
		name  string
		input map[string]any
		want  []string
	}{
		{"all predicates absent", map[string]any{}, []string{"gamma", "beta", "alpha"}},
		{"search OR group", map[string]any{"query": "analytics"}, []string{"gamma", "beta", "alpha"}},
		{"namespace search", map[string]any{"query": "general"}, []string{"gamma", "beta", "alpha"}},
		{"status predicate", map[string]any{"status": "active"}, []string{"alpha"}},
		{"owner predicate", map[string]any{"ownerId": "owner-a"}, []string{"gamma", "alpha"}},
		{"connector predicate", map[string]any{"connectorName": "archive"}, []string{"beta"}},
		{"combined groups", map[string]any{"query": "analytics", "ownerId": "owner-a", "status": "draft"}, []string{"gamma"}},
		{"explicit empty filter is absent", map[string]any{"status": ""}, []string{"gamma", "beta", "alpha"}},
		{"pagination", map[string]any{"limit": 1, "offset": 1}, []string{"beta"}},
		{"ignored legacy selectors", map[string]any{"fields": []string{"slug"}, "orderBy": "slug DESC"}, []string{"gamma", "beta", "alpha"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			output, err := invoke(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if actual := slugs(output); !reflect.DeepEqual(actual, test.want) {
				t.Fatalf("input=%v slugs=%v want=%v", test.input, actual, test.want)
			}
		})
	}

	t.Run("SDK page and package shape", func(t *testing.T) {
		output, err := invoke(map[string]any{"limit": 1, "offset": 1})
		if err != nil {
			t.Fatal(err)
		}
		if output.PageLimit != 1 || output.PageOffset != 1 || len(output.Items) != 1 ||
			output.Items[0].OwnerPackage != "ownerb" || output.Items[0].Title == "" {
			t.Fatalf("SDK report page=%+v", output)
		}
	})
	t.Run("SDK default and capped limit", func(t *testing.T) {
		for _, test := range []struct {
			input map[string]any
			want  int
		}{{map[string]any{}, 50}, {map[string]any{"limit": 999}, 500}} {
			output, err := invoke(test.input)
			if err != nil || output.PageLimit != test.want {
				t.Fatalf("input=%v page=%+v err=%v", test.input, output, err)
			}
		}
	})
	t.Run("SDK HTTP OpenAPI and MCP share the report contract", func(t *testing.T) {
		handler := gateway.NewHandler(runtime, nil, "test")
		request := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/reports.list", bytes.NewBufferString(`{"limit":1,"offset":1}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", jwt.Bearer(t, "viewer"))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("HTTP status=%d body=%s", response.Code, response.Body.String())
		}
		var wire struct {
			Items []struct {
				Slug         string `json:"slug"`
				OwnerPackage string `json:"ownerPackage"`
			} `json:"items"`
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &wire); err != nil || wire.Limit != 1 || wire.Offset != 1 ||
			len(wire.Items) != 1 || wire.Items[0].Slug != "beta" || wire.Items[0].OwnerPackage != "ownerb" {
			t.Fatalf("HTTP page=%+v err=%v body=%s", wire, err, response.Body.String())
		}
		missingAuth := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/reports.list", bytes.NewBufferString(`{}`))
		missingAuth.Header.Set("Content-Type", "application/json")
		missingResponse := httptest.NewRecorder()
		handler.ServeHTTP(missingResponse, missingAuth)
		if missingResponse.Code != http.StatusUnauthorized {
			t.Fatalf("missing JWT status=%d body=%s", missingResponse.Code, missingResponse.Body.String())
		}
		invalid := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/reports.list", bytes.NewBufferString(`{"limit":"invalid"}`))
		invalid.Header.Set("Content-Type", "application/json")
		invalid.Header.Set("Authorization", jwt.Bearer(t, "viewer"))
		invalidResponse := httptest.NewRecorder()
		handler.ServeHTTP(invalidResponse, invalid)
		if invalidResponse.Code < http.StatusBadRequest || invalidResponse.Code >= http.StatusInternalServerError {
			t.Fatalf("invalid page limit status=%d body=%s", invalidResponse.Code, invalidResponse.Body.String())
		}
		document, err := (openapi.Generator{}).Generate(ctx, openapi.Request{
			Info:       openapi3.Info{Title: "Studio SDK", Version: "1"},
			Components: []*registry.RegisteredComponent{authEntry, entry},
			Routes:     []spec.RouteRef{{Method: http.MethodPost, Path: "/v1/studio/sdk/reports.list"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		operation := document.Paths["/v1/studio/sdk/reports.list"].Post
		if operation == nil || operation.RequestBody == nil || operation.Responses == nil || operation.Security == nil {
			t.Fatalf("generated OpenAPI report operation=%+v", operation)
		}
		openAPIWire, err := json.Marshal(operation.RequestBody)
		if err != nil || bytes.Contains(openAPIWire, []byte(`"subject"`)) || bytes.Contains(openAPIWire, []byte(`"scoped"`)) {
			t.Fatalf("OpenAPI exposes trusted report scope: %s err=%v", openAPIWire, err)
		}
		plan, ok := mcpService.Catalog().Tool("studio.sdk.reports.list")
		if !ok {
			t.Fatal("report SDK MCP plan is missing")
		}
		for _, argument := range plan.Arguments() {
			if argument.PublicName() == "subject" || argument.PublicName() == "scoped" || argument.PublicName() == "Auth" || argument.PublicName() == "Jwt" {
				t.Fatalf("MCP exposes trusted report scope: %+v", argument)
			}
		}
		tool, ok := mcpService.Registry().ToolRegistry.Get("studio.sdk.reports.list")
		if !ok {
			t.Fatal("report SDK MCP tool is missing")
		}
		callContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, "viewer")})
		result, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall,
			Params: schema.CallToolRequestParams{Name: "studio.sdk.reports.list", Arguments: map[string]any{"limit": 1, "offset": 1}}})
		if rpcErr != nil || result == nil || result.IsError != nil && *result.IsError {
			t.Fatalf("MCP page=%+v err=%v", result, rpcErr)
		}
		body, err := json.Marshal(result.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		var mcpWire map[string]any
		if err = json.Unmarshal(body, &mcpWire); err != nil || mcpWire["limit"] != float64(1) || mcpWire["offset"] != float64(1) {
			t.Fatalf("MCP page=%s err=%v", body, err)
		}
		mcpItems, ok := mcpWire["items"].([]any)
		if !ok || len(mcpItems) != 1 {
			t.Fatalf("MCP items=%s", body)
		}
		mcpReport, ok := mcpItems[0].(map[string]any)
		if !ok || mcpReport["slug"] != "beta" || mcpReport["ownerPackage"] != "ownerb" {
			t.Fatalf("MCP report=%s", body)
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
		output, err := invokeAs("owner-a", map[string]any{})
		if err != nil || !reflect.DeepEqual(slugs(output), []string{"gamma", "alpha"}) {
			t.Fatalf("owner-a reports=%v err=%v", slugs(output), err)
		}
		output, err = invokeAs("owner-b", map[string]any{})
		if err != nil || !reflect.DeepEqual(slugs(output), []string{"beta"}) {
			t.Fatalf("owner-b reports=%v err=%v", slugs(output), err)
		}
		if _, err = db.ExecContext(ctx, "DELETE FROM report_acl WHERE report_id = ? AND subject_id = ?", "r-beta", "viewer"); err != nil {
			t.Fatal(err)
		}
		output, err = invoke(map[string]any{})
		if err != nil || !reflect.DeepEqual(slugs(output), []string{"gamma", "alpha"}) {
			t.Fatalf("revoked viewer reports=%v err=%v", slugs(output), err)
		}
	})
}
