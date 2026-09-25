package get

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	xresponse "github.com/viant/xdatly/response"
)

func TestNamespaceGetSDKDatlyHTTPMCPAndOpenAPI(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "sdk_namespace_get", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{
			"name": "main", "driver": "sqlite", "owner_id": "bob", "status": "active", "etag": 1,
			"created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00",
		}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{
			{"owner_id": "alice", "name": "general", "title": "General", "status": "active", "etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"},
			{"owner_id": "bob", "name": "finance", "title": "Finance", "description": "Forecasting", "status": "active", "etag": 4, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"},
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
		t.Fatal("generated namespace-get component is missing")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatalf("component metadata: %v", err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name,
		PackageName: "get", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(
		reflect.TypeFor[NamespaceGetInput](), reflect.TypeFor[NamespaceGetOutput]())
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(NamespaceDatlyResourceNamespace, NamespaceDatlyResources); err != nil {
		t.Fatal(err)
	}
	authEntry := datatest.AuthRegistration(t, db, jwt.Factory, resources)
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: reflect.TypeFor[NamespaceGetInput](), OutputType: reflect.TypeFor[NamespaceGetOutput](),
		Resources: resources, Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwt.Factory})
	if err != nil {
		t.Fatal(err)
	}
	connector := &dsql.SQLComponent{DB: db}
	if err = connector.RegisterConnector("studio", db); err != nil {
		t.Fatal(err)
	}
	execution, err := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: connector})
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
	if names := service.Catalog().ToolNames(); !reflect.DeepEqual(names, []string{"studio.sdk.namespaces.get"}) {
		t.Fatalf("MCP tools=%v", names)
	}
	invoke := func(subject, name string) (*NamespaceGetOutput, error) {
		request := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/namespaces.get", bytes.NewBufferString(`{"name":"`+name+`"}`))
		request.Header.Set("Content-Type", "application/json")
		if subject != "" {
			request.Header.Set("Authorization", jwt.Bearer(t, subject))
		}
		scope, scopeErr := requestprovider.New(request)
		if scopeErr != nil {
			t.Fatal(scopeErr)
		}
		defer scope.Close()
		value, invokeErr := runtime.ExecuteRoute(ctx, http.MethodPost, "/v1/studio/sdk/namespaces.get", scope)
		if invokeErr != nil {
			return nil, invokeErr
		}
		return value.(*NamespaceGetOutput), nil
	}
	for _, check := range []struct {
		subject, name, owner string
	}{{"alice", "general", "alice"}, {"bob", "finance", "bob"}, {"viewer", "finance", "bob"}} {
		output, invokeErr := invoke(check.subject, check.name)
		if invokeErr != nil || output == nil || output.ResponseName != check.name || output.OwnerId != check.owner {
			t.Fatalf("%s/%s namespace=%+v err=%v", check.subject, check.name, output, invokeErr)
		}
	}
	for _, check := range []struct{ subject, name string }{{"alice", "finance"}, {"bob", "missing"}} {
		_, invokeErr := invoke(check.subject, check.name)
		var notFound *xresponse.Error
		if !errors.As(invokeErr, &notFound) || notFound.Code != http.StatusNotFound {
			t.Fatalf("%s/%s error=%v, want 404", check.subject, check.name, invokeErr)
		}
	}
	if _, err = invoke("", "finance"); err == nil {
		t.Fatal("missing JWT was accepted")
	}
	handler := gateway.NewHandler(runtime, nil, "test")
	request := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/namespaces.get", bytes.NewBufferString(`{"name":"finance"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", jwt.Bearer(t, "viewer"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d body=%s", response.Code, response.Body.String())
	}
	var wire map[string]any
	if err = json.Unmarshal(response.Body.Bytes(), &wire); err != nil || wire["name"] != "finance" ||
		wire["ownerId"] != "bob" || wire["description"] != "Forecasting" || wire["etag"] != float64(4) || wire["item"] != nil {
		t.Fatalf("HTTP namespace=%v err=%v body=%s", wire, err, response.Body.String())
	}
	for _, check := range []struct {
		subject, name string
		status        int
	}{{"alice", "finance", 404}, {"bob", "missing", 404}, {"", "finance", 401}} {
		denied := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/namespaces.get", bytes.NewBufferString(`{"name":"`+check.name+`"}`))
		denied.Header.Set("Content-Type", "application/json")
		if check.subject != "" {
			denied.Header.Set("Authorization", jwt.Bearer(t, check.subject))
		}
		deniedResponse := httptest.NewRecorder()
		handler.ServeHTTP(deniedResponse, denied)
		if deniedResponse.Code != check.status {
			t.Fatalf("HTTP %s/%s status=%d body=%s", check.subject, check.name, deniedResponse.Code, deniedResponse.Body.String())
		}
	}
	document, err := (openapi.Generator{}).Generate(ctx, openapi.Request{
		Info:       openapi3.Info{Title: "Studio SDK", Version: "1"},
		Components: []*registry.RegisteredComponent{authEntry, entry},
		Routes:     []spec.RouteRef{{Method: http.MethodPost, Path: "/v1/studio/sdk/namespaces.get"}},
	})
	if err != nil || document.Paths["/v1/studio/sdk/namespaces.get"].Post == nil {
		t.Fatalf("namespace OpenAPI=%+v err=%v", document, err)
	}
	tool, ok := service.Registry().ToolRegistry.Get("studio.sdk.namespaces.get")
	if !ok {
		t.Fatal("namespace MCP tool is missing")
	}
	callContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, "viewer")})
	result, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall,
		Params: schema.CallToolRequestParams{Name: "studio.sdk.namespaces.get", Arguments: map[string]any{"name": "finance"}}})
	if rpcErr != nil || result == nil || result.IsError != nil && *result.IsError {
		t.Fatalf("MCP namespace=%+v err=%v", result, rpcErr)
	}
	structured, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var mcpWire map[string]any
	if err = json.Unmarshal(structured, &mcpWire); err != nil || mcpWire["name"] != "finance" || mcpWire["ownerId"] != "bob" {
		t.Fatalf("MCP namespace=%s err=%v", structured, err)
	}
	if _, err = db.ExecContext(ctx, "UPDATE reports SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?", "r1"); err != nil {
		t.Fatal(err)
	}
	_, err = invoke("viewer", "finance")
	var notFound *xresponse.Error
	if !errors.As(err, &notFound) || notFound.Code != http.StatusNotFound {
		t.Fatalf("soft-deleted grant error=%v, want 404", err)
	}
	if _, err = db.ExecContext(ctx, "UPDATE reports SET deleted_at = NULL WHERE id = ?", "r1"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, "DELETE FROM report_acl WHERE report_id = ? AND subject_id = ?", "r1", "viewer"); err != nil {
		t.Fatal(err)
	}
	_, err = invoke("viewer", "finance")
	if !errors.As(err, &notFound) || notFound.Code != http.StatusNotFound {
		t.Fatalf("revoked viewer error=%v, want 404", err)
	}
}
