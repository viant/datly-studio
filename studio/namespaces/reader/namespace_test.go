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
	"github.com/viant/xdatly/response"
)

func TestNamespaceReaderUsesVerifiedTypedScope(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "namespace_read", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{
			"name": "main", "driver": "sqlite", "owner_id": "bob", "status": "active", "etag": 1,
			"created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00",
		}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{
			{"owner_id": "alice", "name": "general", "title": "General", "status": "active", "etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"},
			{"owner_id": "bob", "name": "finance", "title": "Finance", "status": "active", "etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"},
		}},
		datatest.Table{Name: "reports", Rows: []datatest.Row{{
			"id": "r1", "slug": "finance", "title": "Finance", "owner_id": "bob", "namespace": "finance",
			"status": "active", "default_connector_name": "main", "component_scope": "reports/finance", "component_name": "finance",
			"etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00",
		}}},
		datatest.Table{Name: "report_acl", Rows: []datatest.Row{{
			"report_id": "r1", "subject_type": "user", "subject_id": "viewer", "can_view": true,
		}}},
	); err != nil {
		t.Fatal(err)
	}
	holder := reflect.TypeFor[NamespaceComponent]()
	field, ok := holder.FieldByName("Contract")
	if !ok {
		t.Fatal("namespace reader route is missing")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatalf("namespace metadata: %v", err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name,
		PackageName: "reader", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(
		reflect.TypeFor[NamespaceQueryInput](), reflect.TypeFor[NamespaceQueryOutput]())
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(NamespaceDatlyResourceNamespace, NamespaceDatlyResources); err != nil {
		t.Fatal(err)
	}
	authEntry := datatest.AuthRegistration(t, db, jwt.Factory, resources)
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: reflect.TypeFor[NamespaceQueryInput](), OutputType: reflect.TypeFor[NamespaceQueryOutput](),
		Resources: resources, Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwt.Factory})
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
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{authEntry, entry}, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })
	service, err := mcp.New(mcp.Config{Components: []*registry.RegisteredComponent{authEntry, entry}, Invoker: runtime, Resources: resources})
	if err != nil {
		t.Fatal(err)
	}
	if names := service.Catalog().ToolNames(); !reflect.DeepEqual(names, []string{"studio.sdk.namespaces.list"}) {
		t.Fatalf("MCP tools=%v", names)
	}
	read := func(subject string, input map[string]any) (*NamespaceQueryOutput, error) {
		payload, marshalErr := json.Marshal(input)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		request := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/namespaces.list", bytes.NewReader(payload))
		request.Header.Set("Content-Type", "application/json")
		if subject != "" {
			request.Header.Set("Authorization", jwt.Bearer(t, subject))
		}
		scope, scopeErr := requestprovider.New(request)
		if scopeErr != nil {
			t.Fatal(scopeErr)
		}
		defer scope.Close()
		value, invokeErr := runtime.ExecuteRoute(ctx, http.MethodPost, "/v1/studio/sdk/namespaces.list", scope)
		if invokeErr != nil {
			return nil, invokeErr
		}
		return value.(*NamespaceQueryOutput), nil
	}
	names := func(output *NamespaceQueryOutput) []string {
		if output == nil {
			return nil
		}
		result := make([]string, 0, len(output.Items))
		for _, row := range output.Items {
			if row != nil {
				result = append(result, row.Name)
			}
		}
		return result
	}
	for _, check := range []struct {
		subject string
		input   map[string]any
		want    []string
	}{{"alice", map[string]any{}, []string{"general"}}, {"bob", map[string]any{}, []string{"finance"}},
		{"viewer", map[string]any{}, []string{"finance"}}, {"viewer", map[string]any{"query": "FINANCE"}, []string{"finance"}},
		{"viewer", map[string]any{"status": "active"}, []string{"finance"}}, {"viewer", map[string]any{"status": ""}, []string{"finance"}}} {
		output, readErr := read(check.subject, check.input)
		if readErr != nil || !reflect.DeepEqual(names(output), check.want) {
			t.Fatalf("%s input=%v namespaces=%v err=%v", check.subject, check.input, names(output), readErr)
		}
	}
	for _, check := range []struct {
		input  map[string]any
		limit  int
		offset int
	}{{map[string]any{}, 50, 0}, {map[string]any{"limit": 999}, 500, 0}, {map[string]any{"limit": 1, "offset": -4}, 1, 0}} {
		output, readErr := read("viewer", check.input)
		if readErr != nil || output.PageLimit != check.limit || output.PageOffset != check.offset {
			t.Fatalf("page input=%v output=%+v err=%v", check.input, output, readErr)
		}
	}
	if _, err = read("", map[string]any{}); err == nil || response.ErrorStatusCode(err, 500) != 401 {
		t.Fatalf("missing JWT error=%v, want 401", err)
	}
	handler := gateway.NewHandler(runtime, nil, "test")
	request := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/namespaces.list", bytes.NewBufferString(`{"limit":1}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", jwt.Bearer(t, "viewer"))
	httpResponse := httptest.NewRecorder()
	handler.ServeHTTP(httpResponse, request)
	if httpResponse.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d body=%s", httpResponse.Code, httpResponse.Body.String())
	}
	var wire struct {
		Items  []map[string]any `json:"items"`
		Limit  int              `json:"limit"`
		Offset int              `json:"offset"`
	}
	if err = json.Unmarshal(httpResponse.Body.Bytes(), &wire); err != nil || wire.Limit != 1 || wire.Offset != 0 ||
		len(wire.Items) != 1 || wire.Items[0]["name"] != "finance" || wire.Items[0]["ownerId"] != "bob" || wire.Items[0]["etag"] != float64(1) {
		t.Fatalf("HTTP page=%+v err=%v body=%s", wire, err, httpResponse.Body.String())
	}
	document, err := (openapi.Generator{}).Generate(ctx, openapi.Request{
		Info:       openapi3.Info{Title: "Studio SDK", Version: "1"},
		Components: []*registry.RegisteredComponent{authEntry, entry},
		Routes:     []spec.RouteRef{{Method: http.MethodPost, Path: "/v1/studio/sdk/namespaces.list"}},
	})
	if err != nil || document.Paths["/v1/studio/sdk/namespaces.list"].Post == nil {
		t.Fatalf("namespace OpenAPI=%+v err=%v", document, err)
	}
	openAPIWire, err := json.Marshal(document)
	if err != nil || bytes.Contains(openAPIWire, []byte(`"subject"`)) || bytes.Contains(openAPIWire, []byte(`"scoped"`)) {
		t.Fatalf("OpenAPI exposes trusted scope: %s err=%v", openAPIWire, err)
	}
	tool, ok := service.Registry().ToolRegistry.Get("studio.sdk.namespaces.list")
	if !ok {
		t.Fatal("namespace MCP tool is missing")
	}
	callContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, "viewer")})
	result, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall,
		Params: schema.CallToolRequestParams{Name: "studio.sdk.namespaces.list", Arguments: map[string]any{"limit": 1}}})
	if rpcErr != nil || result == nil || result.IsError != nil && *result.IsError {
		t.Fatalf("MCP page=%+v err=%v", result, rpcErr)
	}
	structured, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var mcpWire map[string]any
	if err = json.Unmarshal(structured, &mcpWire); err != nil || mcpWire["limit"] != float64(1) {
		t.Fatalf("MCP page=%s err=%v", structured, err)
	}
	mcpItems, ok := mcpWire["items"].([]any)
	if !ok || len(mcpItems) != 1 {
		t.Fatalf("MCP namespace items=%s", structured)
	}
	mcpNamespace, ok := mcpItems[0].(map[string]any)
	if !ok || mcpNamespace["name"] != "finance" || mcpNamespace["ownerId"] != "bob" {
		t.Fatalf("MCP namespace=%s", structured)
	}
	if _, err = db.ExecContext(ctx, "UPDATE reports SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?", "r1"); err != nil {
		t.Fatal(err)
	}
	deleted, err := read("viewer", map[string]any{})
	if err != nil || len(names(deleted)) != 0 {
		t.Fatalf("soft-deleted report namespace=%v err=%v", names(deleted), err)
	}
	if _, err = db.ExecContext(ctx, "UPDATE reports SET deleted_at = NULL WHERE id = ?", "r1"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, "DELETE FROM report_acl WHERE report_id = ? AND subject_id = ?", "r1", "viewer"); err != nil {
		t.Fatal(err)
	}
	revoked, err := read("viewer", map[string]any{})
	if err != nil || len(names(revoked)) != 0 {
		t.Fatalf("revoked viewer namespace=%v err=%v", names(revoked), err)
	}
}
