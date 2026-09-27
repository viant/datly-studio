package download

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/componentarchive"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/sdk"
	resourcefiles "github.com/viant/datly-studio/studio/report_resource_files/store_download"
	budget "github.com/viant/datly-studio/studio/report_resource_files/store_download_budget"
	versions "github.com/viant/datly-studio/studio/report_versions/reader"
	"github.com/viant/datly/bootstrap"
	gateway "github.com/viant/datly/gateway/http"
	"github.com/viant/datly/gateway/openapi"
	"github.com/viant/datly/gateway/openapi/openapi3"
	"github.com/viant/datly/mcp"
	druntime "github.com/viant/datly/runtime"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	dtag "github.com/viant/datly/tag"
	"github.com/viant/mcp-protocol/authorization"
	"github.com/viant/mcp-protocol/schema"
)

func TestDownloadSDKDatlyHTTPMCPAndOpenAPI(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "sdk_version_download", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{"name": "main", "driver": "sqlite", "owner_id": "alice", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{{"owner_id": "alice", "name": "general", "title": "General", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "components", Rows: []datatest.Row{{"id": "r1", "slug": "first", "title": "First", "owner_id": "alice", "status": "active", "default_connector_name": "main", "namespace": "general", "component_scope": "reports/first", "component_name": "first", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "report_acl", Rows: []datatest.Row{{"report_id": "r1", "subject_type": "user", "subject_id": "viewer", "can_view": true}, {"report_id": "r1", "subject_type": "user", "subject_id": "author", "can_view": true, "can_use_dql": true}}},
		datatest.Table{Name: "report_versions", Rows: []datatest.Row{{"report_id": "r1", "version_no": 1, "state": "draft", "authoring_mode": "dql", "authored_dql": "SELECT 1", "generated_dql": "SELECT 1", "component_spec_json": "{}", "type_manifest_json": "{}", "spec_format_version": "1", "spec_hash": "hash-1", "compile_status": "valid", "datly_version": "v1", "compiler_version": "v1", "source_revision": 1, "created_by": "alice", "created_at": "2026-09-17 09:00:00"}}},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO report_resource_files
 (report_id,version_no,resource_id,namespace,resource_path,content,content_size,content_sha256,is_binary,created_at)
 VALUES(?,?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP)`, "r1", 1, strings.Repeat("a", 64), "assets", "assets/readme.txt", []byte("hello"), 5, strings.Repeat("b", 64), false); err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	for _, registration := range []struct {
		name string
		fs   fs.FS
	}{
		{versions.VersionDatlyResourceNamespace, versions.VersionDatlyResources},
		{resourcefiles.FileDatlyResourceNamespace, resourcefiles.FileDatlyResources},
		{budget.BudgetDatlyResourceNamespace, budget.BudgetDatlyResources},
	} {
		if err := resources.Register(registration.name, registration.fs); err != nil {
			t.Fatal(err)
		}
	}
	connector := &dsql.SQLComponent{DB: db}
	if err := connector.RegisterConnector("studio", db); err != nil {
		t.Fatal(err)
	}
	compile := func(holder reflect.Type, fieldName string, inputType, outputType reflect.Type, handler rhandler.TypedHandler) *registry.RegisteredComponent {
		t.Helper()
		field, ok := holder.FieldByName(fieldName)
		if !ok {
			t.Fatalf("%s.%s is missing", holder, fieldName)
		}
		metadata, present, parseErr := dtag.ParseComponent(field.Tag)
		if parseErr != nil || !present {
			t.Fatalf("%s metadata: %v", holder, parseErr)
		}
		component, resolveErr := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackageName: "download", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(inputType, outputType)
		if resolveErr != nil {
			t.Fatal(resolveErr)
		}
		artifact, buildErr := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: inputType, OutputType: outputType,
			Resources: resources, Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwt.Factory,
			Handler: handler, HandlerOwnedOutput: handler != nil})
		if buildErr != nil {
			t.Fatal(buildErr)
		}
		registration := registry.RegisteredComponent{}
		if handler == nil {
			execution, compileErr := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: connector})
			if compileErr != nil {
				t.Fatal(compileErr)
			}
			registration.Reader = execution
		}
		entry, registerErr := artifact.Registration(registration)
		if registerErr != nil {
			t.Fatal(registerErr)
		}
		return entry
	}
	versionEntry := compile(reflect.TypeFor[versions.VersionComponent](), "Contract2", reflect.TypeFor[versions.Input](), reflect.TypeFor[versions.Output](), nil)
	fileEntry := compile(reflect.TypeFor[resourcefiles.FileComponent](), "Contract", reflect.TypeFor[resourcefiles.Input](), reflect.TypeFor[resourcefiles.Output](), nil)
	budgetEntry := compile(reflect.TypeFor[budget.BudgetComponent](), "Contract", reflect.TypeFor[budget.Input](), reflect.TypeFor[budget.Output](), nil)
	if !fileEntry.Component.Routes[0].Internal || !budgetEntry.Component.Routes[0].Internal {
		t.Fatal("resource download child routes must be internal")
	}
	downloadHandler, err := (Component{}).DatlyHandler("NewDownload")()
	if err != nil {
		t.Fatal(err)
	}
	downloadEntry := compile(reflect.TypeFor[Component](), "Contract", reflect.TypeFor[Input](), reflect.TypeFor[Output](), downloadHandler)
	entries := []*registry.RegisteredComponent{versionEntry, budgetEntry, fileEntry, downloadEntry}
	runtime, err := druntime.NewRuntime(entries, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })
	toolService, err := mcp.New(mcp.Config{Components: entries, Invoker: runtime, Resources: resources})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range toolService.Catalog().ToolNames() {
		if strings.Contains(name, "store") {
			t.Fatalf("internal resource reader exposed MCP tool %s", name)
		}
	}
	if _, ok := toolService.Registry().ToolRegistry.Get("studio.sdk.versions.download"); !ok {
		t.Fatal("native download MCP tool is missing")
	}
	request := func(subject string, versionNo int) *http.Request {
		body, _ := json.Marshal(map[string]any{"reportId": "r1", "versionNo": versionNo})
		req := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/versions.download", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if subject != "" {
			req.Header.Set("Authorization", jwt.Bearer(t, subject))
		}
		return req
	}
	httpHandler := gateway.NewHandler(runtime, nil, "test")
	response := httptest.NewRecorder()
	httpHandler.ServeHTTP(response, request("author", 1))
	if response.Code != http.StatusOK {
		t.Fatalf("HTTP download status=%d body=%s", response.Code, response.Body.String())
	}
	var wire sdk.ComponentDownload
	if err := json.Unmarshal(response.Body.Bytes(), &wire); err != nil || wire.Filename != "component-v1.zip" || wire.EntryDQL != "component.dql" {
		t.Fatalf("HTTP download=%+v err=%v body=%s", wire, err, response.Body.String())
	}
	bundle, err := sdk.ReadDQLArchive(bytes.NewReader(wire.Archive), "zip")
	if err != nil || string(bundle.Files[wire.EntryDQL]) != "SELECT 1" || string(bundle.Files["assets/readme.txt"]) != "hello" {
		t.Fatalf("archive=%v err=%v", bundle, err)
	}
	for _, check := range []struct {
		subject string
		version int
		status  int
	}{{"viewer", 1, 404}, {"intruder", 1, 404}, {"author", 999, 404}, {"", 1, 401}} {
		denied := httptest.NewRecorder()
		httpHandler.ServeHTTP(denied, request(check.subject, check.version))
		if denied.Code != check.status || bytes.Contains(denied.Body.Bytes(), []byte("hello")) {
			t.Fatalf("%s/%d status=%d body=%s", check.subject, check.version, denied.Code, denied.Body.String())
		}
	}
	internal := httptest.NewRecorder()
	httpHandler.ServeHTTP(internal, httptest.NewRequest(http.MethodGet, "/_studio/report-resource-files-store/download?reportId=r1&versionNo=1", nil))
	if internal.Code == http.StatusOK {
		t.Fatalf("internal resource reader is publicly reachable: %s", internal.Body.String())
	}
	internal = httptest.NewRecorder()
	httpHandler.ServeHTTP(internal, httptest.NewRequest(http.MethodGet, "/_studio/report-resource-files-store/download-budget?reportId=r1&versionNo=1", nil))
	if internal.Code == http.StatusOK {
		t.Fatalf("internal resource budget is publicly reachable: %s", internal.Body.String())
	}
	document, err := (openapi.Generator{}).Generate(ctx, openapi.Request{Info: openapi3.Info{Title: "Studio SDK", Version: "1"}, Components: entries, Routes: []spec.RouteRef{{Method: "POST", Path: "/v1/studio/sdk/versions.download"}}})
	if err != nil || document.Paths["/v1/studio/sdk/versions.download"].Post == nil || document.Paths["/_studio/report-resource-files-store/download"] != nil || document.Paths["/_studio/report-resource-files-store/download-budget"] != nil {
		t.Fatalf("OpenAPI download/internal routes: %v", err)
	}
	tool, _ := toolService.Registry().ToolRegistry.Get("studio.sdk.versions.download")
	call := func(subject string) ([]byte, error) {
		callContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, subject)})
		result, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall, Params: schema.CallToolRequestParams{Name: "studio.sdk.versions.download", Arguments: map[string]any{"reportId": "r1", "versionNo": 1}}})
		if rpcErr != nil {
			return nil, rpcErr
		}
		if result.IsError != nil && *result.IsError {
			return nil, nil
		}
		return json.Marshal(result.StructuredContent)
	}
	structured, err := call("author")
	if err != nil || !bytes.Contains(structured, []byte("component-v1.zip")) {
		t.Fatalf("MCP download=%s err=%v", structured, err)
	}
	structured, err = call("viewer")
	if err != nil || bytes.Contains(structured, []byte("component-v1.zip")) {
		t.Fatalf("MCP viewer archive leaked=%s err=%v", structured, err)
	}
	large := bytes.Repeat([]byte("x"), componentarchive.MaxResourceBytes+1)
	if _, err = db.ExecContext(ctx, `INSERT INTO report_resource_files
 (report_id,version_no,resource_id,namespace,resource_path,content,content_size,content_sha256,is_binary,created_at)
 VALUES(?,?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP)`, "r1", 1, strings.Repeat("c", 64), "assets", "assets/oversized.bin", large, len(large), strings.Repeat("d", 64), true); err != nil {
		t.Fatal(err)
	}
	oversized := httptest.NewRecorder()
	httpHandler.ServeHTTP(oversized, request("author", 1))
	if oversized.Code != http.StatusBadRequest || bytes.Contains(oversized.Body.Bytes(), large[:64]) {
		t.Fatalf("oversized archive status=%d body=%s", oversized.Code, oversized.Body.String())
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM report_acl WHERE report_id = ? AND subject_id = ?", "r1", "author"); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	httpHandler.ServeHTTP(response, request("author", 1))
	if response.Code != http.StatusNotFound {
		t.Fatalf("revoked DQL grant status=%d body=%s", response.Code, response.Body.String())
	}
}
