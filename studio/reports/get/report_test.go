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
	"github.com/viant/datly-studio/internal/namespaceaccess"
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

func TestReportGetSDKDatlyHTTPMCPAndOpenAPI(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "sdk_report_get", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{
			"name": "main", "driver": "sqlite", "owner_id": "alice", "status": "active",
			"etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00",
		}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{{
			"owner_id": "alice", "name": "general", "title": "General", "status": "active",
			"etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00",
		}}},
		datatest.Table{Name: "components", Rows: []datatest.Row{{
			"id": "r1", "slug": "first", "title": "First", "owner_id": "alice",
			"status": "active", "default_connector_name": "main", "namespace": "general",
			"component_scope": "reports/first", "component_name": "first",
			"etag": 9, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00",
		}}},
		datatest.Table{Name: "component_acl", Rows: []datatest.Row{{
			"report_id": "r1", "subject_type": "user", "subject_id": "bob", "can_view": true,
		}}},
	); err != nil {
		t.Fatal(err)
	}
	holder := reflect.TypeFor[ReportComponent]()
	field, ok := holder.FieldByName("Contract")
	if !ok {
		t.Fatal("generated report-get component is missing")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatalf("component metadata: %v", err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name,
		PackageName: "get", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(
		reflect.TypeFor[ReportGetInput](), reflect.TypeFor[ReportGetOutput]())
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(ReportDatlyResourceNamespace, ReportDatlyResources); err != nil {
		t.Fatal(err)
	}
	authEntry := datatest.AuthRegistration(t, db, jwt.Factory, resources)
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: reflect.TypeFor[ReportGetInput](), OutputType: reflect.TypeFor[ReportGetOutput](),
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
	entry.Capabilities.Connector = connector
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{authEntry, entry}, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })
	toolService, err := mcp.New(mcp.Config{Components: []*registry.RegisteredComponent{authEntry, entry}, Invoker: runtime, Resources: resources})
	if err != nil {
		t.Fatal(err)
	}
	if names := toolService.Catalog().ToolNames(); !reflect.DeepEqual(names, []string{"studio.sdk.components.get"}) {
		t.Fatalf("MCP tools=%v", names)
	}
	invoke := func(subject, id string) (*ReportGetOutput, error) {
		request := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/components.get", bytes.NewBufferString(`{"id":"`+id+`"}`))
		request.Header.Set("Content-Type", "application/json")
		if subject != "" {
			request.Header.Set("Authorization", jwt.Bearer(t, subject))
		}
		scope, scopeErr := requestprovider.New(request)
		if scopeErr != nil {
			t.Fatal(scopeErr)
		}
		defer scope.Close()
		value, invokeErr := runtime.ExecuteRoute(ctx, http.MethodPost, "/v1/studio/sdk/components.get", scope)
		if invokeErr != nil {
			return nil, invokeErr
		}
		return value.(*ReportGetOutput), nil
	}
	for _, subject := range []string{"alice", "bob"} {
		output, invokeErr := invoke(subject, "r1")
		if invokeErr != nil || output == nil || output.Item == nil || output.Item.Id != "r1" ||
			output.Item.OwnerPackage != "alice" || output.Item.Etag != 9 {
			t.Fatalf("%s report=%+v err=%v", subject, output, invokeErr)
		}
	}
	for _, subject := range []string{"charlie", "alice"} {
		id := "r1"
		if subject == "alice" {
			id = "missing"
		}
		_, invokeErr := invoke(subject, id)
		var notFound *xresponse.Error
		if !errors.As(invokeErr, &notFound) || notFound.Code != http.StatusNotFound {
			t.Fatalf("%s report %s error=%v, want 404", subject, id, invokeErr)
		}
	}
	if _, err = invoke("", "r1"); err == nil {
		t.Fatal("missing JWT was accepted")
	}
	handler := gateway.NewHandler(runtime, nil, "test")
	request := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/components.get", bytes.NewBufferString(`{"id":"r1"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", jwt.Bearer(t, "alice"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d body=%s", response.Code, response.Body.String())
	}
	var wire map[string]any
	if err = json.Unmarshal(response.Body.Bytes(), &wire); err != nil || wire["id"] != "r1" ||
		wire["ownerPackage"] != "alice" || wire["etag"] != float64(9) || wire["item"] != nil {
		t.Fatalf("HTTP report=%v err=%v body=%s", wire, err, response.Body.String())
	}
	if _, err := db.ExecContext(ctx, `UPDATE namespaces SET namespace_id=? WHERE owner_id='alice' AND name='general'`, namespaceaccess.ID("alice", "general")); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO namespaces(namespace_id,owner_id,name,title,status,etag,created_at,updated_at) VALUES(?,'alice','other','Other','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, namespaceaccess.ID("alice", "other")); err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct {
		subject, id string
		status      int
	}{
		{"alice", namespaceaccess.ID("alice", "general"), 200},
		{"alice", namespaceaccess.ID("alice", "other"), 404},
		{"bob", namespaceaccess.ID("alice", "general"), 404},
		{"alice", "invalid", 403},
		{"alice", "", 403},
	} {
		scopedRequest := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/components.get", bytes.NewBufferString(`{"id":"r1"}`))
		scopedRequest.Header.Set("Content-Type", "application/json")
		scopedRequest.Header.Set("Authorization", jwt.Bearer(t, check.subject))
		scopedRequest.Header.Set("X-Studio-Namespace", check.id)
		scopedResponse := httptest.NewRecorder()
		handler.ServeHTTP(scopedResponse, scopedRequest)
		if scopedResponse.Code != check.status {
			t.Fatalf("namespace header subject=%s id=%q status=%d want=%d body=%s", check.subject, check.id, scopedResponse.Code, check.status, scopedResponse.Body.String())
		}
	}

	for _, check := range []struct {
		subject string
		id      string
		status  int
	}{{"alice", "missing", http.StatusNotFound}, {"charlie", "r1", http.StatusNotFound}, {"", "r1", http.StatusUnauthorized}} {
		denied := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/components.get", bytes.NewBufferString(`{"id":"`+check.id+`"}`))
		denied.Header.Set("Content-Type", "application/json")
		if check.subject != "" {
			denied.Header.Set("Authorization", jwt.Bearer(t, check.subject))
		}
		deniedResponse := httptest.NewRecorder()
		handler.ServeHTTP(deniedResponse, denied)
		if deniedResponse.Code != check.status || bytes.Contains(deniedResponse.Body.Bytes(), []byte(`"ownerPackage":"alice"`)) {
			t.Fatalf("HTTP %s/%s status=%d body=%s", check.subject, check.id, deniedResponse.Code, deniedResponse.Body.String())
		}
	}
	missingID := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/components.get", bytes.NewBufferString(`{}`))
	missingID.Header.Set("Content-Type", "application/json")
	missingID.Header.Set("Authorization", jwt.Bearer(t, "alice"))
	missingIDResponse := httptest.NewRecorder()
	handler.ServeHTTP(missingIDResponse, missingID)
	if missingIDResponse.Code < http.StatusBadRequest || missingIDResponse.Code >= http.StatusInternalServerError {
		t.Fatalf("missing report id status=%d body=%s", missingIDResponse.Code, missingIDResponse.Body.String())
	}
	document, err := (openapi.Generator{}).Generate(ctx, openapi.Request{
		Info:       openapi3.Info{Title: "Studio SDK", Version: "1"},
		Components: []*registry.RegisteredComponent{authEntry, entry},
		Routes:     []spec.RouteRef{{Method: http.MethodPost, Path: "/v1/studio/sdk/components.get"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if document.Paths["/v1/studio/sdk/components.get"].Post == nil {
		t.Fatal("OpenAPI lacks components.get POST route")
	}
	tool, ok := toolService.Registry().ToolRegistry.Get("studio.sdk.components.get")
	if !ok {
		t.Fatal("MCP report-get tool is missing")
	}

	for _, check := range []struct {
		subject, id string
		allowed     bool
	}{
		{"alice", namespaceaccess.ID("alice", "general"), true},
		{"alice", namespaceaccess.ID("alice", "other"), false},
		{"bob", namespaceaccess.ID("alice", "general"), false},
		{"alice", "invalid", false},
	} {
		scopedContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, check.subject)})
		scopedResult, scopedErr := tool.Handler(scopedContext, &schema.CallToolRequest{Method: schema.MethodToolsCall, Params: schema.CallToolRequestParams{Name: "studio.sdk.components.get", Arguments: map[string]any{"id": "r1", "namespaceId": check.id}}})
		allowed := scopedErr == nil && scopedResult != nil && (scopedResult.IsError == nil || !*scopedResult.IsError)
		if allowed != check.allowed {
			t.Fatalf("MCP namespace selection subject=%s namespace=%s result=%+v err=%v", check.subject, check.id, scopedResult, scopedErr)
		}
	}
	callContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, "bob")})
	result, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall,
		Params: schema.CallToolRequestParams{Name: "studio.sdk.components.get", Arguments: map[string]any{"id": "r1"}}})
	if rpcErr != nil || result == nil || result.IsError != nil && *result.IsError {
		t.Fatalf("MCP report=%+v err=%v", result, rpcErr)
	}
	structured, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var mcpWire map[string]any
	if err = json.Unmarshal(structured, &mcpWire); err != nil || mcpWire["id"] != "r1" || mcpWire["ownerPackage"] != "alice" {
		t.Fatalf("MCP report=%s err=%v", structured, err)
	}
	if _, err = db.ExecContext(ctx, "DELETE FROM component_acl WHERE report_id = ? AND subject_id = ?", "r1", "bob"); err != nil {
		t.Fatal(err)
	}
	_, err = invoke("bob", "r1")
	var notFound *xresponse.Error
	if !errors.As(err, &notFound) || notFound.Code != http.StatusNotFound {
		t.Fatalf("revoked reader error=%v, want 404", err)
	}
	revokedTool, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall,
		Params: schema.CallToolRequestParams{Name: "studio.sdk.components.get", Arguments: map[string]any{"id": "r1"}}})
	if rpcErr == nil && revokedTool != nil && (revokedTool.IsError == nil || !*revokedTool.IsError) {
		t.Fatalf("revoked MCP reader returned data: %+v", revokedTool)
	}
}
