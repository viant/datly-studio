package export_dql

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

func TestVersionExportDQLSDKDatlyHTTPMCPAndOpenAPI(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "sdk_version_export_dql", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{"name": "main", "driver": "sqlite", "owner_id": "alice", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{{"owner_id": "alice", "name": "general", "title": "General", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "components", Rows: []datatest.Row{{"id": "r1", "slug": "first", "title": "First", "owner_id": "alice", "status": "active", "default_connector_name": "main", "namespace": "general", "component_scope": "reports/first", "component_name": "first", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "report_acl", Rows: []datatest.Row{{"report_id": "r1", "subject_type": "user", "subject_id": "viewer", "can_view": true}, {"report_id": "r1", "subject_type": "user", "subject_id": "author", "can_view": true, "can_use_dql": true}}},
		datatest.Table{Name: "report_versions", Rows: []datatest.Row{
			{"report_id": "r1", "version_no": 1, "state": "draft", "authoring_mode": "dql", "authored_sql": "fallback SQL", "authored_dql": "authored DQL", "generated_dql": "generated DQL", "component_spec_json": "{}", "type_manifest_json": "{}", "spec_format_version": "1", "spec_hash": "hash-1", "compile_status": "valid", "datly_version": "v1", "compiler_version": "v1", "source_revision": 1, "created_by": "alice", "created_at": "2026-09-17 09:00:00"},
			{"report_id": "r1", "version_no": 2, "state": "draft", "authoring_mode": "dql", "authored_sql": "fallback SQL", "authored_dql": "authored DQL", "generated_dql": "", "component_spec_json": "{}", "type_manifest_json": "{}", "spec_format_version": "1", "spec_hash": "hash-2", "compile_status": "valid", "datly_version": "v1", "compiler_version": "v1", "source_revision": 1, "created_by": "alice", "created_at": "2026-09-17 09:00:00"},
			{"report_id": "r1", "version_no": 3, "state": "draft", "authoring_mode": "sql", "authored_sql": "fallback SQL", "authored_dql": "", "generated_dql": "", "component_spec_json": "{}", "type_manifest_json": "{}", "spec_format_version": "1", "spec_hash": "hash-3", "compile_status": "valid", "datly_version": "v1", "compiler_version": "v1", "source_revision": 1, "created_by": "alice", "created_at": "2026-09-17 09:00:00"},
			{"report_id": "r1", "version_no": 4, "state": "draft", "authoring_mode": "sql", "authored_sql": "", "authored_dql": "", "generated_dql": "", "component_spec_json": "{}", "type_manifest_json": "{}", "spec_format_version": "1", "spec_hash": "hash-4", "compile_status": "valid", "datly_version": "v1", "compiler_version": "v1", "source_revision": 1, "created_by": "alice", "created_at": "2026-09-17 09:00:00"},
		}},
	); err != nil {
		t.Fatal(err)
	}
	holder := reflect.TypeFor[VersionComponent]()
	field, ok := holder.FieldByName("Contract")
	if !ok {
		t.Fatal("version export component is missing")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatalf("component metadata: %v", err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackageName: "export_dql", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(reflect.TypeFor[VersionExportInput](), reflect.TypeFor[VersionExportOutput]())
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(VersionDatlyResourceNamespace, VersionDatlyResources); err != nil {
		t.Fatal(err)
	}
	authEntry := datatest.AuthRegistration(t, db, jwt.Factory, resources)
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeFor[VersionExportInput](), OutputType: reflect.TypeFor[VersionExportOutput](), Resources: resources, Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwt.Factory})
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
	if names := tools.Catalog().ToolNames(); !reflect.DeepEqual(names, []string{"studio.sdk.versions.export_dql"}) {
		t.Fatalf("MCP tools=%v", names)
	}
	request := func(subject string, versionNo int) *http.Request {
		body, _ := json.Marshal(map[string]any{"reportId": "r1", "versionNo": versionNo})
		req := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/versions.export_dql", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if subject != "" {
			req.Header.Set("Authorization", jwt.Bearer(t, subject))
		}
		return req
	}
	handler := gateway.NewHandler(runtime, nil, "test")
	for _, check := range []struct {
		versionNo int
		dql       string
		complete  bool
	}{{1, "generated DQL", true}, {2, "authored DQL", true}, {3, "fallback SQL", true}, {4, "", false}} {
		scope, scopeErr := requestprovider.New(request("author", check.versionNo))
		if scopeErr != nil {
			t.Fatal(scopeErr)
		}
		value, invokeErr := runtime.ExecuteRoute(ctx, http.MethodPost, "/v1/studio/sdk/versions.export_dql", scope)
		scope.Close()
		if invokeErr != nil {
			t.Fatalf("version %d: %v", check.versionNo, invokeErr)
		}
		output := value.(*VersionExportOutput)
		if output.Dql != check.dql || output.Complete != check.complete {
			t.Fatalf("version %d output=%+v", check.versionNo, output)
		}
	}
	for _, check := range []struct {
		subject string
		version int
		status  int
	}{{"viewer", 1, 404}, {"intruder", 1, 404}, {"author", 999, 404}, {"", 1, 401}} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request(check.subject, check.version))
		if response.Code != check.status || bytes.Contains(response.Body.Bytes(), []byte("DQL")) {
			t.Fatalf("%s/%d status=%d body=%s", check.subject, check.version, response.Code, response.Body.String())
		}
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request("author", 1))
	if response.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d body=%s", response.Code, response.Body.String())
	}
	var wire map[string]any
	if err = json.Unmarshal(response.Body.Bytes(), &wire); err != nil || wire["dql"] != "generated DQL" || wire["complete"] != true || wire["item"] != nil {
		t.Fatalf("HTTP export=%v err=%v body=%s", wire, err, response.Body.String())
	}
	document, err := (openapi.Generator{}).Generate(ctx, openapi.Request{Info: openapi3.Info{Title: "Studio SDK", Version: "1"}, Components: []*registry.RegisteredComponent{authEntry, entry}, Routes: []spec.RouteRef{{Method: http.MethodPost, Path: "/v1/studio/sdk/versions.export_dql"}}})
	if err != nil || document.Paths["/v1/studio/sdk/versions.export_dql"].Post == nil {
		t.Fatalf("OpenAPI export route: %v", err)
	}
	tool, ok := tools.Registry().ToolRegistry.Get("studio.sdk.versions.export_dql")
	if !ok {
		t.Fatal("MCP export tool is missing")
	}
	call := func(subject string) ([]byte, error) {
		callContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, subject)})
		result, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall, Params: schema.CallToolRequestParams{Name: "studio.sdk.versions.export_dql", Arguments: map[string]any{"reportId": "r1", "versionNo": 1}}})
		if rpcErr != nil {
			return nil, rpcErr
		}
		if result.IsError != nil && *result.IsError {
			return nil, nil
		}
		return json.Marshal(result.StructuredContent)
	}
	structured, err := call("author")
	if err != nil || !bytes.Contains(structured, []byte("generated DQL")) {
		t.Fatalf("MCP export=%s err=%v", structured, err)
	}
	structured, err = call("viewer")
	if err != nil || bytes.Contains(structured, []byte("generated DQL")) {
		t.Fatalf("MCP viewer source leaked: %s err=%v", structured, err)
	}
	if _, err = db.ExecContext(ctx, "DELETE FROM report_acl WHERE report_id = ? AND subject_id = ?", "r1", "author"); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request("author", 1))
	if response.Code != http.StatusNotFound {
		t.Fatalf("revoked DQL grant status=%d body=%s", response.Code, response.Body.String())
	}
}
