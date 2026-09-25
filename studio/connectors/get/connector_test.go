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

func TestConnectorGetSDKDatlyHTTPMCPAndOpenAPI(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "sdk_connector_get", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{
			"name": "main", "driver": "sqlite", "dsn_template": "file:private-token", "secret_ref": "secret://private-token",
			"owner_id": "alice", "status": "active", "options_json": `{}`, "etag": 7,
			"created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00",
		}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{{
			"owner_id": "alice", "name": "general", "title": "General", "status": "active",
			"etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00",
		}}},
		datatest.Table{Name: "reports", Rows: []datatest.Row{{
			"id": "r1", "slug": "first", "title": "First", "owner_id": "alice",
			"status": "active", "default_connector_name": "main", "namespace": "general",
			"component_scope": "reports/first", "component_name": "first",
			"etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00",
		}}},
		datatest.Table{Name: "report_acl", Rows: []datatest.Row{{
			"report_id": "r1", "subject_type": "user", "subject_id": "bob", "can_view": true,
		}}},
	); err != nil {
		t.Fatal(err)
	}
	holder := reflect.TypeFor[ConnectorComponent]()
	field, ok := holder.FieldByName("Contract")
	if !ok {
		t.Fatal("generated connector-get component is missing")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatalf("component metadata: %v", err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name,
		PackageName: "get", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(
		reflect.TypeFor[ConnectorGetInput](), reflect.TypeFor[ConnectorGetOutput]())
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(ConnectorDatlyResourceNamespace, ConnectorDatlyResources); err != nil {
		t.Fatal(err)
	}
	authEntry := datatest.AuthRegistration(t, db, jwt.Factory, resources)
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: reflect.TypeFor[ConnectorGetInput](), OutputType: reflect.TypeFor[ConnectorGetOutput](),
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
	toolService, err := mcp.New(mcp.Config{Components: []*registry.RegisteredComponent{authEntry, entry}, Invoker: runtime, Resources: resources})
	if err != nil {
		t.Fatal(err)
	}
	if names := toolService.Catalog().ToolNames(); !reflect.DeepEqual(names, []string{"studio.sdk.connectors.get"}) {
		t.Fatalf("MCP tools=%v", names)
	}
	invoke := func(subject, name string) (*ConnectorGetOutput, error) {
		request := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/connectors.get", bytes.NewBufferString(`{"name":"`+name+`"}`))
		request.Header.Set("Content-Type", "application/json")
		if subject != "" {
			request.Header.Set("Authorization", jwt.Bearer(t, subject))
		}
		scope, scopeErr := requestprovider.New(request)
		if scopeErr != nil {
			t.Fatal(scopeErr)
		}
		defer scope.Close()
		value, invokeErr := runtime.ExecuteRoute(ctx, http.MethodPost, "/v1/studio/sdk/connectors.get", scope)
		if invokeErr != nil {
			return nil, invokeErr
		}
		return value.(*ConnectorGetOutput), nil
	}
	for _, subject := range []string{"alice", "bob"} {
		output, invokeErr := invoke(subject, "main")
		if invokeErr != nil || output == nil || output.ResponseName != "main" || !output.DsnConfigured ||
			!output.SecretConfigured || output.Etag != 7 || string(output.Options) != `{}` {
			t.Fatalf("%s connector=%+v err=%v", subject, output, invokeErr)
		}
	}
	for _, check := range []struct{ subject, name string }{{"charlie", "main"}, {"alice", "missing"}} {
		_, invokeErr := invoke(check.subject, check.name)
		var notFound *xresponse.Error
		if !errors.As(invokeErr, &notFound) || notFound.Code != http.StatusNotFound {
			t.Fatalf("%s/%s error=%v, want 404", check.subject, check.name, invokeErr)
		}
	}
	if _, err = invoke("", "main"); err == nil {
		t.Fatal("missing JWT was accepted")
	}
	handler := gateway.NewHandler(runtime, nil, "test")
	request := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/connectors.get", bytes.NewBufferString(`{"name":"main"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", jwt.Bearer(t, "alice"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d body=%s", response.Code, response.Body.String())
	}
	var wire map[string]any
	if err = json.Unmarshal(response.Body.Bytes(), &wire); err != nil || wire["name"] != "main" ||
		wire["dsnConfigured"] != true || wire["secretConfigured"] != true || wire["etag"] != float64(7) ||
		wire["options"] == nil || wire["item"] != nil {
		t.Fatalf("HTTP connector=%v err=%v body=%s", wire, err, response.Body.String())
	}
	if bytes.Contains(response.Body.Bytes(), []byte("private-token")) || bytes.Contains(response.Body.Bytes(), []byte("dsnTemplate")) || bytes.Contains(response.Body.Bytes(), []byte("secretRef")) {
		t.Fatalf("HTTP connector leaked secret material: %s", response.Body.String())
	}
	for _, check := range []struct {
		subject, name string
		status        int
	}{{"alice", "missing", 404}, {"charlie", "main", 404}, {"", "main", 401}} {
		denied := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/connectors.get", bytes.NewBufferString(`{"name":"`+check.name+`"}`))
		denied.Header.Set("Content-Type", "application/json")
		if check.subject != "" {
			denied.Header.Set("Authorization", jwt.Bearer(t, check.subject))
		}
		deniedResponse := httptest.NewRecorder()
		handler.ServeHTTP(deniedResponse, denied)
		if deniedResponse.Code != check.status || bytes.Contains(deniedResponse.Body.Bytes(), []byte("private-token")) {
			t.Fatalf("HTTP %s/%s status=%d body=%s", check.subject, check.name, deniedResponse.Code, deniedResponse.Body.String())
		}
	}
	document, err := (openapi.Generator{}).Generate(ctx, openapi.Request{
		Info:       openapi3.Info{Title: "Studio SDK", Version: "1"},
		Components: []*registry.RegisteredComponent{authEntry, entry},
		Routes:     []spec.RouteRef{{Method: http.MethodPost, Path: "/v1/studio/sdk/connectors.get"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if document.Paths["/v1/studio/sdk/connectors.get"].Post == nil {
		t.Fatal("OpenAPI lacks connectors.get POST route")
	}
	openAPIWire, err := json.Marshal(document)
	if err != nil || bytes.Contains(openAPIWire, []byte(`"dsnTemplate"`)) || bytes.Contains(openAPIWire, []byte(`"secretRef"`)) || bytes.Contains(openAPIWire, []byte(`"subject"`)) {
		t.Fatalf("OpenAPI exposes secrets or trusted scope: %s err=%v", openAPIWire, err)
	}
	tool, ok := toolService.Registry().ToolRegistry.Get("studio.sdk.connectors.get")
	if !ok {
		t.Fatal("MCP connector-get tool is missing")
	}
	callContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, "bob")})
	result, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall,
		Params: schema.CallToolRequestParams{Name: "studio.sdk.connectors.get", Arguments: map[string]any{"name": "main"}}})
	if rpcErr != nil || result == nil || result.IsError != nil && *result.IsError {
		t.Fatalf("MCP connector=%+v err=%v", result, rpcErr)
	}
	structured, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var mcpWire map[string]any
	if err = json.Unmarshal(structured, &mcpWire); err != nil || mcpWire["name"] != "main" || mcpWire["dsnConfigured"] != true || mcpWire["secretConfigured"] != true {
		t.Fatalf("MCP connector=%s err=%v", structured, err)
	}
	if bytes.Contains(structured, []byte("private-token")) || bytes.Contains(structured, []byte("dsnTemplate")) {
		t.Fatalf("MCP connector leaked secret material: %s", structured)
	}
	if _, err = db.ExecContext(ctx, "UPDATE reports SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?", "r1"); err != nil {
		t.Fatal(err)
	}
	_, err = invoke("bob", "main")
	var notFound *xresponse.Error
	if !errors.As(err, &notFound) || notFound.Code != http.StatusNotFound {
		t.Fatalf("soft-deleted grant error=%v, want 404", err)
	}
	if _, err = db.ExecContext(ctx, "UPDATE reports SET deleted_at = NULL WHERE id = ?", "r1"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, "DELETE FROM report_acl WHERE report_id = ? AND subject_id = ?", "r1", "bob"); err != nil {
		t.Fatal(err)
	}
	_, err = invoke("bob", "main")
	if !errors.As(err, &notFound) || notFound.Code != http.StatusNotFound {
		t.Fatalf("revoked viewer error=%v, want 404", err)
	}
}
