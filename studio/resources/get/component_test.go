package get

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/sdk"
	files "github.com/viant/datly-studio/studio/report_resource_files/store_snapshot"
	folders "github.com/viant/datly-studio/studio/report_resource_folders/store_snapshot"
	skills "github.com/viant/datly-studio/studio/report_skill_roots/store_snapshot"
	versionget "github.com/viant/datly-studio/studio/report_versions/get"
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

func TestResourceGetSDKDatlyHTTPMCPAndOpenAPI(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "sdk_resources_get", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{"name": "main", "driver": "sqlite", "owner_id": "alice", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{{"owner_id": "alice", "name": "general", "title": "General", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "components", Rows: []datatest.Row{{"id": "r1", "slug": "first", "title": "First", "owner_id": "alice", "status": "active", "default_connector_name": "main", "namespace": "general", "component_scope": "reports/first", "component_name": "first", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "report_acl", Rows: []datatest.Row{{"report_id": "r1", "subject_type": "user", "subject_id": "viewer", "can_view": true}}},
		datatest.Table{Name: "report_versions", Rows: []datatest.Row{{"report_id": "r1", "version_no": 2, "state": "draft", "authoring_mode": "dql", "authored_dql": "private source", "generated_dql": "private generated source", "component_spec_json": "{}", "type_manifest_json": "{}", "spec_format_version": "1", "spec_hash": "hash-2", "compile_status": "valid", "datly_version": "v1", "compiler_version": "v1", "source_revision": 2, "created_by": "alice", "created_at": "2026-09-17 09:00:00"}}},
	); err != nil {
		t.Fatal(err)
	}
	const count = 137
	for i := count - 1; i >= 0; i-- {
		id, path := fmt.Sprintf("%064x", i+1), fmt.Sprintf("assets/file-%03d.txt", i)
		if _, err := db.ExecContext(ctx, `INSERT INTO report_resource_files
 (report_id,version_no,resource_id,namespace,resource_path,content,content_size,content_sha256,is_binary,created_at)
 VALUES(?,?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP)`, "r1", 2, id, "assets", path, []byte(fmt.Sprintf("value-%03d", i)), 9, id, false); err != nil {
			t.Fatal(err)
		}
	}
	for _, item := range []struct {
		id, root string
		ordinal  int
	}{{"b", "second", 2}, {"a", "first", 1}} {
		if _, err := db.ExecContext(ctx, `INSERT INTO report_resource_folders
 (report_id,version_no,folder_id,namespace,root_path,uri_prefix,ordinal)
 VALUES(?,?,?,?,?,?,?)`, "r1", 2, item.id, "assets", item.root, "skill://"+item.root+"/", item.ordinal); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO report_skill_roots
 (report_id,version_no,skill_id,folder_id,skill_root,ordinal)
 VALUES(?,?,?,?,?,?)`, "r1", 2, item.id, item.id, item.root, item.ordinal); err != nil {
			t.Fatal(err)
		}
	}
	resources := resource.New()
	for _, item := range []struct {
		name string
		fs   fs.FS
	}{
		{versionget.VersionDatlyResourceNamespace, versionget.VersionDatlyResources},
		{files.FileDatlyResourceNamespace, files.FileDatlyResources},
		{folders.FolderDatlyResourceNamespace, folders.FolderDatlyResources},
		{skills.SkillDatlyResourceNamespace, skills.SkillDatlyResources},
	} {
		if err := resources.Register(item.name, item.fs); err != nil {
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
		component, resolveErr := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackageName: "get", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(inputType, outputType)
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
	authEntry := datatest.AuthRegistration(t, db, jwt.Factory, resources)
	versionEntry := compile(reflect.TypeFor[versionget.VersionComponent](), "Contract", reflect.TypeFor[versionget.VersionGetInput](), reflect.TypeFor[versionget.VersionGetOutput](), nil)
	fileEntry := compile(reflect.TypeFor[files.FileComponent](), "Contract", reflect.TypeFor[files.Input](), reflect.TypeFor[files.Output](), nil)
	folderEntry := compile(reflect.TypeFor[folders.FolderComponent](), "Contract", reflect.TypeFor[folders.Input](), reflect.TypeFor[folders.Output](), nil)
	skillEntry := compile(reflect.TypeFor[skills.SkillComponent](), "Contract", reflect.TypeFor[skills.Input](), reflect.TypeFor[skills.Output](), nil)
	for _, entry := range []*registry.RegisteredComponent{fileEntry, folderEntry, skillEntry} {
		if !entry.Component.Routes[0].Internal || len(entry.Component.Routes[0].MCP) != 0 {
			t.Fatalf("snapshot child is public: %+v", entry.Component.Routes[0])
		}
	}
	snapshotHandler, err := (Component{}).DatlyHandler("NewResourceSnapshot")()
	if err != nil {
		t.Fatal(err)
	}
	snapshotEntry := compile(reflect.TypeFor[Component](), "Contract", reflect.TypeFor[Input](), reflect.TypeFor[Output](), snapshotHandler)
	entries := []*registry.RegisteredComponent{authEntry, versionEntry, fileEntry, folderEntry, skillEntry, snapshotEntry}
	runtime, err := druntime.NewRuntime(entries, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })
	tools, err := mcp.New(mcp.Config{Components: entries, Invoker: runtime, Resources: resources})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := tools.Registry().ToolRegistry.Get("studio.sdk.resources.get"); !ok {
		t.Fatal("native resource snapshot MCP tool is missing")
	}
	for _, name := range tools.Catalog().ToolNames() {
		if strings.Contains(name, "snapshot") {
			t.Fatalf("private snapshot reader exposed MCP tool %s", name)
		}
	}
	request := func(subject string, versionNo int) *http.Request {
		body, _ := json.Marshal(map[string]any{"reportId": "r1", "versionNo": versionNo})
		req := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/resources.get", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if subject != "" {
			req.Header.Set("Authorization", jwt.Bearer(t, subject))
		}
		return req
	}
	handler := gateway.NewHandler(runtime, nil, "test")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request("viewer", 2))
	if response.Code != http.StatusOK {
		t.Fatalf("HTTP snapshot status=%d body=%s", response.Code, response.Body.String())
	}
	var wire sdk.ResourceSnapshot
	if err := json.Unmarshal(response.Body.Bytes(), &wire); err != nil || len(wire.Files) != count || len(wire.Folders) != 2 || len(wire.Skills) != 2 || wire.Version == nil {
		t.Fatalf("HTTP snapshot files/folders/skills=%d/%d/%d version=%+v err=%v", len(wire.Files), len(wire.Folders), len(wire.Skills), wire.Version, err)
	}
	if wire.Files[0].ResourcePath != "assets/file-000.txt" || wire.Files[count-1].ResourcePath != "assets/file-136.txt" || wire.Folders[0].FolderID != "a" || wire.Skills[0].SkillID != "a" || wire.Version.GeneratedDQL != "" || wire.Version.AuthoredDQL != "" {
		t.Fatalf("snapshot ordering/redaction: first=%+v last=%+v folders=%+v skills=%+v version=%+v", wire.Files[0], wire.Files[count-1], wire.Folders, wire.Skills, wire.Version)
	}
	for _, check := range []struct {
		subject string
		version int
		status  int
	}{{"intruder", 2, 404}, {"viewer", 999, 404}, {"", 2, 401}} {
		denied := httptest.NewRecorder()
		handler.ServeHTTP(denied, request(check.subject, check.version))
		if denied.Code != check.status || bytes.Contains(denied.Body.Bytes(), []byte("value-000")) {
			t.Fatalf("%s/%d status=%d body=%s", check.subject, check.version, denied.Code, denied.Body.String())
		}
	}
	for _, path := range []string{"/_studio/resource-snapshot/files", "/_studio/resource-snapshot/folders", "/_studio/resource-snapshot/skills"} {
		internal := httptest.NewRecorder()
		handler.ServeHTTP(internal, httptest.NewRequest(http.MethodGet, path+"?reportId=r1&versionNo=2", nil))
		if internal.Code == http.StatusOK {
			t.Fatalf("internal snapshot reader %s is publicly reachable", path)
		}
	}
	document, err := (openapi.Generator{}).Generate(ctx, openapi.Request{Info: openapi3.Info{Title: "Studio SDK", Version: "1"}, Components: entries, Routes: []spec.RouteRef{{Method: "POST", Path: "/v1/studio/sdk/resources.get"}}})
	if err != nil || document.Paths["/v1/studio/sdk/resources.get"].Post == nil {
		t.Fatalf("OpenAPI resource snapshot: %v", err)
	}
	for _, path := range []string{"/_studio/resource-snapshot/files", "/_studio/resource-snapshot/folders", "/_studio/resource-snapshot/skills"} {
		if document.Paths[path] != nil {
			t.Fatalf("internal snapshot reader %s appears in OpenAPI", path)
		}
	}
	tool, _ := tools.Registry().ToolRegistry.Get("studio.sdk.resources.get")
	call := func(subject string) ([]byte, error) {
		callContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, subject)})
		result, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall, Params: schema.CallToolRequestParams{Name: "studio.sdk.resources.get", Arguments: map[string]any{"reportId": "r1", "versionNo": 2}}})
		if rpcErr != nil {
			return nil, rpcErr
		}
		if result.IsError != nil && *result.IsError {
			return nil, nil
		}
		return json.Marshal(result.StructuredContent)
	}
	structured, err := call("viewer")
	if err != nil || !bytes.Contains(structured, []byte("assets/file-136.txt")) || bytes.Contains(structured, []byte("private source")) {
		t.Fatalf("MCP snapshot=%s err=%v", structured, err)
	}
	structured, err = call("intruder")
	if err != nil || bytes.Contains(structured, []byte("assets/file-136.txt")) {
		t.Fatalf("MCP intruder snapshot leaked=%s err=%v", structured, err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM report_acl WHERE report_id = ? AND subject_id = ?", "r1", "viewer"); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request("viewer", 2))
	if response.Code != http.StatusNotFound {
		t.Fatalf("revoked view grant status=%d body=%s", response.Code, response.Body.String())
	}
}
