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

func TestPublicationGetSDKDatlyHTTPMCPAndOpenAPI(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "sdk_publication_get", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{"name": "main", "driver": "sqlite", "owner_id": "alice", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{{"owner_id": "alice", "name": "general", "title": "General", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "reports", Rows: []datatest.Row{{"id": "r1", "slug": "first", "title": "First", "owner_id": "alice", "status": "active", "default_connector_name": "main", "namespace": "general", "component_scope": "reports/first", "component_name": "first", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "report_acl", Rows: []datatest.Row{{"report_id": "r1", "subject_type": "user", "subject_id": "bob", "can_view": true}}},
		datatest.Table{Name: "report_versions", Rows: []datatest.Row{{"report_id": "r1", "version_no": 1, "state": "published", "authoring_mode": "dql", "component_spec_json": "{}", "spec_format_version": "1", "spec_hash": "hash", "type_manifest_json": "{}", "compile_status": "valid", "datly_version": "v1", "compiler_version": "v1", "source_revision": 1, "created_by": "alice", "created_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "runtime_generations", Rows: []datatest.Row{{"generation_no": 7, "source_revision": "r1:1:7", "status": "active", "report_count": 1, "build_manifest_json": "{}", "requested_by": "alice", "requested_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "report_publications", Rows: []datatest.Row{{"report_id": "r1", "active_version_no": 1, "desired_version_no": 1, "desired_generation": 7, "active_generation": 7, "publication_status": "active", "runtime_revision": "r1:1:7", "spec_hash": "hash", "published_by": "alice", "published_at": "2026-09-17 10:00:00"}}},
	); err != nil {
		t.Fatal(err)
	}
	holder := reflect.TypeFor[PublicationComponent]()
	field, ok := holder.FieldByName("Contract")
	if !ok {
		t.Fatal("generated publication-get component is missing")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatalf("component metadata: %v", err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name,
		PackageName: "get", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(
		reflect.TypeFor[PublicationGetInput](), reflect.TypeFor[PublicationGetOutput]())
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(PublicationDatlyResourceNamespace, PublicationDatlyResources); err != nil {
		t.Fatal(err)
	}
	authEntry := datatest.AuthRegistration(t, db, jwt.Factory, resources)
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: reflect.TypeFor[PublicationGetInput](), OutputType: reflect.TypeFor[PublicationGetOutput](),
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
	tools, err := mcp.New(mcp.Config{Components: []*registry.RegisteredComponent{authEntry, entry}, Invoker: runtime, Resources: resources})
	if err != nil {
		t.Fatal(err)
	}
	if names := tools.Catalog().ToolNames(); !reflect.DeepEqual(names, []string{"studio.sdk.publications.get"}) {
		t.Fatalf("MCP tools=%v", names)
	}
	invoke := func(subject, id string) (*PublicationGetOutput, error) {
		request := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/publications.get", bytes.NewBufferString(`{"reportId":"`+id+`"}`))
		request.Header.Set("Content-Type", "application/json")
		if subject != "" {
			request.Header.Set("Authorization", jwt.Bearer(t, subject))
		}
		scope, scopeErr := requestprovider.New(request)
		if scopeErr != nil {
			t.Fatal(scopeErr)
		}
		defer scope.Close()
		value, invokeErr := runtime.ExecuteRoute(ctx, http.MethodPost, "/v1/studio/sdk/publications.get", scope)
		if invokeErr != nil {
			return nil, invokeErr
		}
		return value.(*PublicationGetOutput), nil
	}
	for _, subject := range []string{"alice", "bob"} {
		output, invokeErr := invoke(subject, "r1")
		if invokeErr != nil || output.ResponseReportId != "r1" || output.ActiveVersionNo != 1 ||
			output.DesiredGeneration != 7 || output.ActiveGeneration == nil || *output.ActiveGeneration != 7 || output.Status != "active" {
			t.Fatalf("%s publication=%+v err=%v", subject, output, invokeErr)
		}
	}
	for _, check := range []struct{ subject, id string }{{"charlie", "r1"}, {"alice", "missing"}} {
		_, invokeErr := invoke(check.subject, check.id)
		var notFound *xresponse.Error
		if !errors.As(invokeErr, &notFound) || notFound.Code != http.StatusNotFound {
			t.Fatalf("%s/%s error=%v, want 404", check.subject, check.id, invokeErr)
		}
	}
	if _, err = invoke("", "r1"); err == nil {
		t.Fatal("missing JWT was accepted")
	}
	handler := gateway.NewHandler(runtime, nil, "test")
	request := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/publications.get", bytes.NewBufferString(`{"reportId":"r1"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", jwt.Bearer(t, "alice"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d body=%s", response.Code, response.Body.String())
	}
	var wire map[string]any
	if err = json.Unmarshal(response.Body.Bytes(), &wire); err != nil || len(wire) != 9 || wire["reportId"] != "r1" ||
		wire["activeVersionNo"] != float64(1) || wire["desiredVersionNo"] != float64(1) ||
		wire["desiredGeneration"] != float64(7) || wire["activeGeneration"] != float64(7) ||
		wire["status"] != "active" || wire["runtimeRevision"] != "r1:1:7" ||
		wire["specHash"] != "hash" || wire["publishedAt"] == nil || wire["item"] != nil {
		t.Fatalf("HTTP publication=%v err=%v body=%s", wire, err, response.Body.String())
	}
	for _, check := range []struct {
		subject, id string
		status      int
	}{{"alice", "missing", http.StatusNotFound}, {"charlie", "r1", http.StatusNotFound}, {"", "r1", http.StatusUnauthorized}} {
		denied := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/publications.get", bytes.NewBufferString(`{"reportId":"`+check.id+`"}`))
		denied.Header.Set("Content-Type", "application/json")
		if check.subject != "" {
			denied.Header.Set("Authorization", jwt.Bearer(t, check.subject))
		}
		deniedResponse := httptest.NewRecorder()
		handler.ServeHTTP(deniedResponse, denied)
		if deniedResponse.Code != check.status || bytes.Contains(deniedResponse.Body.Bytes(), []byte(`"reportId":"r1"`)) {
			t.Fatalf("HTTP %s/%s status=%d body=%s", check.subject, check.id, deniedResponse.Code, deniedResponse.Body.String())
		}
	}
	document, err := (openapi.Generator{}).Generate(ctx, openapi.Request{
		Info:       openapi3.Info{Title: "Studio SDK", Version: "1"},
		Components: []*registry.RegisteredComponent{authEntry, entry},
		Routes:     []spec.RouteRef{{Method: http.MethodPost, Path: "/v1/studio/sdk/publications.get"}},
	})
	if err != nil || document.Paths["/v1/studio/sdk/publications.get"].Post == nil {
		t.Fatalf("OpenAPI publication route missing: %v", err)
	}
	tool, ok := tools.Registry().ToolRegistry.Get("studio.sdk.publications.get")
	if !ok {
		t.Fatal("MCP publication-get tool is missing")
	}
	callContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, "bob")})
	result, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall,
		Params: schema.CallToolRequestParams{Name: "studio.sdk.publications.get", Arguments: map[string]any{"reportId": "r1"}}})
	if rpcErr != nil || result == nil || result.IsError != nil && *result.IsError {
		t.Fatalf("MCP publication=%+v err=%v", result, rpcErr)
	}
	structured, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var mcpWire map[string]any
	if err = json.Unmarshal(structured, &mcpWire); err != nil || mcpWire["reportId"] != "r1" || mcpWire["activeGeneration"] != float64(7) {
		t.Fatalf("MCP publication=%s err=%v", structured, err)
	}
	if _, err = db.ExecContext(ctx, "DELETE FROM report_acl WHERE report_id = ? AND subject_id = ?", "r1", "bob"); err != nil {
		t.Fatal(err)
	}
	_, err = invoke("bob", "r1")
	var notFound *xresponse.Error
	if !errors.As(err, &notFound) || notFound.Code != http.StatusNotFound {
		t.Fatalf("revoked publication read error=%v, want 404", err)
	}
	revokedTool, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall,
		Params: schema.CallToolRequestParams{Name: "studio.sdk.publications.get", Arguments: map[string]any{"reportId": "r1"}}})
	if rpcErr == nil && revokedTool != nil && (revokedTool.IsError == nil || !*revokedTool.IsError) {
		t.Fatalf("revoked MCP publication returned data: %+v", revokedTool)
	}
}
