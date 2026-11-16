package execute

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"embed"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/internal/readercomponent"
	connectorcatalog "github.com/viant/datly-studio/studio/connectors/store_catalog"
	versionget "github.com/viant/datly-studio/studio/report_versions/get"
	inspection "github.com/viant/datly-studio/studio/report_versions/inspect"
	versioncatalog "github.com/viant/datly-studio/studio/report_versions/store_catalog"
	reportget "github.com/viant/datly-studio/studio/reports/get"
	capabilities "github.com/viant/datly-studio/studio/reports/store_capabilities"
	runaccess "github.com/viant/datly-studio/studio/reports/store_run_access"
	"github.com/viant/datly/bootstrap"
	gateway "github.com/viant/datly/gateway/http"
	"github.com/viant/datly/mcp"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
	dtag "github.com/viant/datly/tag"
	"github.com/viant/mcp-protocol/authorization"
	"github.com/viant/mcp-protocol/schema"
)

func TestPreviewArtifactsNativeHTTPAndMCP(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	db := datatest.OpenSQLite(t, "preview_artifacts", "studio")
	dsn := "file:" + filepath.Join(t.TempDir(), "business.db")
	business, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer business.Close()
	if _, err = business.ExecContext(ctx, `CREATE TABLE records(id INTEGER PRIMARY KEY,name TEXT);INSERT INTO records VALUES(1,'current')`); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	source := `#package('github.com/viant/datly-studio/proof/records')
#setting($_ = $connector('main'))
#setting($_ = $route('/records','GET'))
#define($_ = $Rows<[]*Record>(output/view))
SELECT records.*,type(records,'Record') FROM (SELECT * FROM records) records`
	if err = datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{"name": "main", "driver": "sqlite", "dsn_template": dsn, "owner_id": "owner", "status": "active", "created_at": now, "updated_at": now}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{{"owner_id": "owner", "name": "general", "title": "General", "status": "active", "created_at": now, "updated_at": now}}},
		datatest.Table{Name: "components", Rows: []datatest.Row{{"id": "reader", "slug": "reader", "title": "Reader", "owner_id": "owner", "status": "draft", "default_connector_name": "main", "namespace": "general", "component_scope": "github.com/viant/datly-studio/proof/records", "component_name": "reader", "created_at": now, "updated_at": now}}},
		datatest.Table{Name: "component_versions", Rows: []datatest.Row{{"report_id": "reader", "version_no": 1, "state": "draft", "authoring_mode": "dql", "authored_dql": source, "generated_dql": source, "component_spec_json": "{}", "type_manifest_json": "{}", "spec_format_version": "1", "spec_hash": "hash", "compile_status": "valid", "datly_version": "v1", "compiler_version": "v1", "source_revision": 7, "created_by": "owner", "created_at": now}}},
	); err != nil {
		t.Fatal(err)
	}
	jwt := datatest.NewJWTFixture(t)
	resources := resource.New()
	connector := &dsql.SQLComponent{DB: db}
	if err = connector.RegisterConnector("studio", db); err != nil {
		t.Fatal(err)
	}
	if err = resources.Register(runaccess.AccessDatlyResourceNamespace, runaccess.AccessDatlyResources); err != nil {
		t.Fatal(err)
	}
	access, _, err := readercomponent.Compile(reflect.TypeFor[runaccess.AccessComponent](), "access", reflect.TypeFor[runaccess.Input](), reflect.TypeFor[runaccess.Output](), resources, connector)
	if err != nil {
		t.Fatal(err)
	}
	holder := reflect.TypeFor[Component]()
	field, _ := holder.FieldByName("Contract")
	metadata, _, err := dtag.ParseComponent(field.Tag)
	if err != nil {
		t.Fatal(err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(reflect.TypeFor[Input](), reflect.TypeFor[Output]())
	if err != nil {
		t.Fatal(err)
	}
	handler, err := (Component{}).DatlyHandler("NewExecute")()
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeFor[Input](), OutputType: reflect.TypeFor[Output](), Resources: resources, Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwt.Factory, Handler: handler, HandlerOwnedOutput: true})
	if err != nil {
		t.Fatal(err)
	}
	entry, err := artifact.Registration(registry.RegisteredComponent{})
	if err != nil {
		t.Fatal(err)
	}
	entries := []*registry.RegisteredComponent{datatest.AuthRegistration(t, db, jwt.Factory, resources), access, entry}

	// Compile the real inspection dependencies, including its source-redacting
	// public version reader. Inspection must commit its artifact before returning.
	type embeddedHolder interface {
		EmbedFS() *embed.FS
		EmbedNamespace() string
	}
	for _, row := range []struct {
		holder  embeddedHolder
		in, out reflect.Type
	}{
		{versionget.VersionComponent{}, reflect.TypeFor[versionget.VersionGetInput](), reflect.TypeFor[versionget.VersionGetOutput]()},
		{versioncatalog.VersionComponent{}, reflect.TypeFor[versioncatalog.Input](), reflect.TypeFor[versioncatalog.Output]()},
		{reportget.ReportComponent{}, reflect.TypeFor[reportget.ReportGetInput](), reflect.TypeFor[reportget.ReportGetOutput]()},
		{capabilities.CapabilityComponent{}, reflect.TypeFor[capabilities.Input](), reflect.TypeFor[capabilities.Output]()},
		{connectorcatalog.ConnectorComponent{}, reflect.TypeFor[connectorcatalog.Input](), reflect.TypeFor[connectorcatalog.Output]()},
	} {
		if err = resources.Register(row.holder.EmbedNamespace(), row.holder.EmbedFS()); err != nil {
			t.Fatal(err)
		}
		typ := reflect.TypeOf(row.holder)
		field, _ := typ.FieldByName("Contract")
		tag, _, err := dtag.ParseComponent(field.Tag)
		if err != nil {
			t.Fatal(err)
		}
		spec, err := (&bootstrap.RouteSource{HolderType: typ.Name(), FieldName: field.Name, PackagePath: typ.PkgPath(), Tag: tag}).Resolve(row.in, row.out)
		if err != nil {
			t.Fatal(err)
		}
		artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: spec, InputType: row.in, OutputType: row.out, Resources: resources, Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwt.Factory})
		if err != nil {
			t.Fatal(err)
		}
		reader, err := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: connector})
		if err != nil {
			t.Fatal(err)
		}
		entry, err := artifact.Registration(registry.RegisteredComponent{Reader: reader})
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, entry)
	}
	inspectHolder := reflect.TypeFor[inspection.Component]()
	inspectField, _ := inspectHolder.FieldByName("Contract")
	inspectTag, _, err := dtag.ParseComponent(inspectField.Tag)
	if err != nil {
		t.Fatal(err)
	}
	inspectSpec, err := (&bootstrap.RouteSource{HolderType: inspectHolder.Name(), FieldName: inspectField.Name, PackagePath: inspectHolder.PkgPath(), Tag: inspectTag}).Resolve(reflect.TypeFor[inspection.Input](), reflect.TypeFor[inspection.Output]())
	if err != nil {
		t.Fatal(err)
	}
	inspectHandler, err := (inspection.Component{}).DatlyHandler("NewVersionInspect")()
	if err != nil {
		t.Fatal(err)
	}
	inspectArtifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: inspectSpec, InputType: reflect.TypeFor[inspection.Input](), OutputType: reflect.TypeFor[inspection.Output](), Resources: resources, Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwt.Factory, Handler: inspectHandler, HandlerOwnedOutput: true})
	if err != nil {
		t.Fatal(err)
	}
	inspectEntry, err := inspectArtifact.Registration(registry.RegisteredComponent{})
	if err != nil {
		t.Fatal(err)
	}
	entries = append(entries, inspectEntry)

	for _, entry := range entries {
		entry.Capabilities.Connector = connector
	}
	runtime, err := druntime.NewRuntime(entries, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Shutdown(context.WithoutCancel(ctx))
	httpHandler := gateway.NewHandler(runtime, nil, "test")
	request := func(subject string, refresh bool) *httptest.ResponseRecorder {
		t.Helper()
		raw, _ := json.Marshal(map[string]any{"reportId": "reader", "versionNo": 1, "input": map[string]any{"refreshSchema": refresh}})
		req := httptest.NewRequestWithContext(ctx, "POST", "/v1/studio/sdk/preview.execute", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", jwt.Bearer(t, subject))
		response := httptest.NewRecorder()
		httpHandler.ServeHTTP(response, req)
		return response
	}
	inspectedRequest := httptest.NewRequestWithContext(ctx, "POST", "/v1/studio/sdk/versions.inspect", strings.NewReader(`{"reportId":"reader","versionNo":1,"discoverColumns":false,"includePreviewInput":true,"includeEffectiveWarmup":false}`))
	inspectedRequest.Header.Set("Content-Type", "application/json")
	inspectedRequest.Header.Set("Authorization", jwt.Bearer(t, "owner"))
	inspected := httptest.NewRecorder()
	httpHandler.ServeHTTP(inspected, inspectedRequest)
	if inspected.Code != 200 || !strings.Contains(inspected.Body.String(), `"previewInput"`) {
		t.Fatalf("native input inspection: %d %s", inspected.Code, inspected.Body.String())
	}
	first := request("owner", false)
	if first.Code != http.StatusOK {
		t.Fatalf("first preview: %d %s", first.Code, first.Body.String())
	}
	if _, err = business.ExecContext(ctx, `ALTER TABLE records ADD COLUMN extra TEXT;UPDATE records SET extra='discovered-column'`); err != nil {
		t.Fatal(err)
	}
	reused := request("owner", false)
	if reused.Code != 200 || strings.Contains(reused.Body.String(), "discovered-column") {
		t.Fatalf("ordinary hit: %d %s", reused.Code, reused.Body.String())
	}
	denied := request("outsider", true)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("denied refresh: %d %s", denied.Code, denied.Body.String())
	}
	tools, err := mcp.New(mcp.Config{Components: entries, Invoker: runtime, Resources: resources})
	if err != nil {
		t.Fatal(err)
	}
	tool, found := tools.Registry().ToolRegistry.Get("studio.sdk.preview.execute")
	if !found {
		t.Fatal("preview MCP tool missing")
	}
	callCtx := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, "owner")})
	result, rpcErr := tool.Handler(callCtx, &schema.CallToolRequest{Method: schema.MethodToolsCall, Params: schema.CallToolRequestParams{Name: "studio.sdk.preview.execute", Arguments: map[string]any{"reportId": "reader", "versionNo": 1, "input": map[string]any{"refreshSchema": true}}}})
	if rpcErr != nil || result == nil || (result.IsError != nil && *result.IsError) {
		t.Fatalf("MCP refresh: %+v %v", result, rpcErr)
	}
	payload, err := json.Marshal(result.StructuredContent)
	if err != nil || !bytes.Contains(payload, []byte("discovered-column")) {
		t.Fatalf("MCP did not refresh schema: %s %v", payload, err)
	}
	if again := request("owner", true); again.Code != 200 {
		t.Fatalf("identical refresh: %d %s", again.Code, again.Body.String())
	}
	var revision, count int
	if err = db.QueryRowContext(ctx, `SELECT source_revision FROM component_versions WHERE report_id='reader' AND version_no=1`).Scan(&revision); err != nil || revision != 7 {
		t.Fatalf("changed source: %d %v", revision, err)
	}
	if err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_resource_files WHERE report_id='reader' AND namespace='__datly_compiled'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("preview artifact count=%d err=%v", count, err)
	}
}
