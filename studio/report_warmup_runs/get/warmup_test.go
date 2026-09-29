package get

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	requestprovider "github.com/viant/bindly/provider/request"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/sdk"
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

func TestWarmupGetSDKDatlyHTTPMCPAndOpenAPI(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "sdk_warmup_get", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{"name": "main", "driver": "sqlite", "owner_id": "alice", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{{"owner_id": "alice", "name": "general", "title": "General", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "components", Rows: []datatest.Row{{"id": "r1", "slug": "first", "title": "First", "owner_id": "alice", "status": "active", "default_connector_name": "main", "namespace": "general", "component_scope": "reports/first", "component_name": "first", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "report_versions", Rows: []datatest.Row{{"report_id": "r1", "version_no": 2, "state": "draft", "authoring_mode": "dql", "component_spec_json": "{}", "type_manifest_json": "{}", "spec_format_version": "1", "spec_hash": "hash", "compile_status": "valid", "datly_version": "v1", "compiler_version": "v1", "source_revision": 3, "created_by": "alice", "created_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "report_acl", Rows: []datatest.Row{{"report_id": "r1", "subject_type": "user", "subject_id": "publisher", "can_view": true, "can_publish": true}, {"report_id": "r1", "subject_type": "user", "subject_id": "viewer", "can_view": true}}},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO report_warmup_runs
 (run_id,report_id,version_no,source_revision,spec_hash,plan_key,status,requested_by,
 requested_at,created_at,created_by,updated_at,updated_by,started_at,completed_at,
 planned_cases,completed_cases,max_cases,row_limit,entries,duration_ns,target_json,diagnostics_json)
 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"w1", "r1", 2, 3, "hash", "plan", "partial", "alice",
		"2026-09-17 09:00:00", "2026-09-17 09:00:00", "alice", "2026-09-17 09:01:00", "alice",
		"2026-09-17 09:00:05", "2026-09-17 09:01:00", 7, 5, 10, 20, 3, 123456,
		`{"view":"reader","cacheName":"primary"}`, `[ {"severity":"error","code":"warmup_failed","message":"partial"} ]`); err != nil {
		t.Fatal(err)
	}
	holder := reflect.TypeFor[WarmupRunComponent]()
	field, ok := holder.FieldByName("Contract")
	if !ok {
		t.Fatal("warmup-get component is missing")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatalf("component metadata: %v", err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name,
		PackageName: "get", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(
		reflect.TypeFor[WarmupGetInput](), reflect.TypeFor[WarmupGetOutput]())
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(WarmupRunDatlyResourceNamespace, WarmupRunDatlyResources); err != nil {
		t.Fatal(err)
	}
	authEntry := datatest.AuthRegistration(t, db, jwt.Factory, resources)
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: reflect.TypeFor[WarmupGetInput](), OutputType: reflect.TypeFor[WarmupGetOutput](),
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
	tools, err := mcp.New(mcp.Config{Components: []*registry.RegisteredComponent{authEntry, entry}, Invoker: runtime, Resources: resources})
	if err != nil {
		t.Fatal(err)
	}
	if names := tools.Catalog().ToolNames(); !reflect.DeepEqual(names, []string{"studio.sdk.versions.warmup_get"}) {
		t.Fatalf("MCP tools=%v", names)
	}
	request := func(subject, reportID, runID string) *http.Request {
		body, _ := json.Marshal(map[string]any{"reportId": reportID, "runId": runID})
		req := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/versions.warmup_get", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if subject != "" {
			req.Header.Set("Authorization", jwt.Bearer(t, subject))
		}
		return req
	}
	otherID := namespaceaccess.ID("alice", "other")
	if _, err := db.ExecContext(ctx, `INSERT INTO namespaces(owner_id,name,title,status,namespace_id,created_at,updated_at) VALUES('alice','other','Other','active',?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, otherID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE namespaces SET namespace_id=? WHERE owner_id='alice' AND name='general'`, namespaceaccess.ID("alice", "general")); err != nil {
		t.Fatal(err)
	}
	selectedRequest := request("alice", "r1", "w1")
	selectedRequest.Header.Set("X-Studio-Namespace", otherID)
	selectedScope, err := requestprovider.New(selectedRequest)
	if err != nil {
		t.Fatal(err)
	}
	selectedValue, selectedErr := runtime.ExecuteRoute(ctx, http.MethodPost, "/v1/studio/sdk/versions.warmup_get", selectedScope)
	selectedScope.Close()
	if selectedErr == nil {
		t.Fatalf("another workspace returned warmup metadata: %+v", selectedValue)
	}
	scope, err := requestprovider.New(request("publisher", "r1", "w1"))
	if err != nil {
		t.Fatal(err)
	}
	value, err := runtime.ExecuteRoute(ctx, http.MethodPost, "/v1/studio/sdk/versions.warmup_get", scope)
	scope.Close()
	if err != nil {
		t.Fatal(err)
	}
	output := value.(*WarmupGetOutput)
	if output.Response == nil || output.Response.RunID != "w1" || output.Response.Target.CacheName != "primary" || len(output.Response.Diagnostics) != 1 || output.Response.Duration != time.Duration(123456) {
		t.Fatalf("warmup response=%+v", output.Response)
	}
	handler := gateway.NewHandler(runtime, nil, "test")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request("publisher", "r1", "w1"))
	if response.Code != http.StatusOK || bytes.Contains(response.Body.Bytes(), []byte(`"item"`)) || bytes.Contains(response.Body.Bytes(), []byte(`"response"`)) {
		t.Fatalf("HTTP warmup status=%d body=%s", response.Code, response.Body.String())
	}
	var wire sdk.WarmupRun
	if err = json.Unmarshal(response.Body.Bytes(), &wire); err != nil || wire.RunID != "w1" || wire.ReportID != "r1" || wire.Target.View != "reader" || wire.Entries != 3 {
		t.Fatalf("HTTP warmup=%+v err=%v body=%s", wire, err, response.Body.String())
	}
	for _, check := range []struct {
		subject, reportID, runID string
		status                   int
	}{{"viewer", "r1", "w1", 404}, {"intruder", "r1", "w1", 404}, {"publisher", "r1", "missing", 404}, {"publisher", "other", "w1", 404}, {"", "r1", "w1", 401}} {
		denied := httptest.NewRecorder()
		handler.ServeHTTP(denied, request(check.subject, check.reportID, check.runID))
		if denied.Code != check.status || bytes.Contains(denied.Body.Bytes(), []byte("primary")) {
			t.Fatalf("%s/%s/%s status=%d body=%s", check.subject, check.reportID, check.runID, denied.Code, denied.Body.String())
		}
	}
	document, err := (openapi.Generator{}).Generate(ctx, openapi.Request{Info: openapi3.Info{Title: "Studio SDK", Version: "1"}, Components: []*registry.RegisteredComponent{authEntry, entry}, Routes: []spec.RouteRef{{Method: "POST", Path: "/v1/studio/sdk/versions.warmup_get"}}})
	if err != nil || document.Paths["/v1/studio/sdk/versions.warmup_get"].Post == nil {
		t.Fatalf("OpenAPI warmup route: %v", err)
	}
	responseSchema := document.Paths["/v1/studio/sdk/versions.warmup_get"].Post.Responses["200"].Content["application/json"].Schema
	if responseSchema == nil || !strings.HasPrefix(responseSchema.Ref, "#/components/schemas/") {
		t.Fatalf("OpenAPI warmup response=%+v", responseSchema)
	}
	tool, ok := tools.Registry().ToolRegistry.Get("studio.sdk.versions.warmup_get")
	if !ok {
		t.Fatal("MCP warmup tool is missing")
	}
	callContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, "publisher")})
	result, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall, Params: schema.CallToolRequestParams{Name: "studio.sdk.versions.warmup_get", Arguments: map[string]any{"reportId": "r1", "runId": "w1"}}})
	if rpcErr != nil || result == nil || result.IsError != nil && *result.IsError {
		t.Fatalf("MCP warmup result=%+v err=%v", result, rpcErr)
	}
	structured, err := json.Marshal(result.StructuredContent)
	if err != nil || !bytes.Contains(structured, []byte(`"cacheName":"primary"`)) {
		t.Fatalf("MCP warmup=%s err=%v", structured, err)
	}
	if _, err = db.ExecContext(ctx, "DELETE FROM report_acl WHERE report_id = ? AND subject_id = ?", "r1", "publisher"); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request("publisher", "r1", "w1"))
	if response.Code != http.StatusNotFound {
		t.Fatalf("revoked publish grant status=%d body=%s", response.Code, response.Body.String())
	}
}
