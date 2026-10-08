package descriptor

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

func TestVersionDescriptorSDKDatlyHTTPMCPAndOpenAPI(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "sdk_version_descriptor", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{"name": "main", "driver": "sqlite", "owner_id": "alice", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{{"owner_id": "alice", "name": "general", "title": "General", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "components", Rows: []datatest.Row{{"id": "r1", "slug": "first", "title": "First", "owner_id": "alice", "status": "active", "default_connector_name": "main", "namespace": "general", "component_scope": "reports/first", "component_name": "first", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "component_acl", Rows: []datatest.Row{{"report_id": "r1", "subject_type": "user", "subject_id": "viewer", "can_view": true}}},
		datatest.Table{Name: "component_versions", Rows: []datatest.Row{{"report_id": "r1", "version_no": 1, "state": "draft", "authoring_mode": "dql", "authored_dql": "source must not leak", "component_spec_json": `{"route":"/x"}`, "type_manifest_json": `{"record":"x"}`, "resource_manifest_json": `{"files":[]}`, "spec_format_version": "1", "spec_hash": "hash-1", "compile_status": "valid", "datly_version": "v1", "compiler_version": "v1", "source_revision": 1, "created_by": "alice", "created_at": "2026-09-17 09:00:00"}}},
	); err != nil {
		t.Fatal(err)
	}
	holder := reflect.TypeFor[VersionComponent]()
	field, ok := holder.FieldByName("Contract")
	if !ok {
		t.Fatal("version descriptor component is missing")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatalf("component metadata: %v", err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackageName: "descriptor", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(reflect.TypeFor[VersionDescriptorInput](), reflect.TypeFor[VersionDescriptorOutput]())
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(VersionDatlyResourceNamespace, VersionDatlyResources); err != nil {
		t.Fatal(err)
	}
	authEntry := datatest.AuthRegistration(t, db, jwt.Factory, resources)
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeFor[VersionDescriptorInput](), OutputType: reflect.TypeFor[VersionDescriptorOutput](), Resources: resources, Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwt.Factory})
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
	if names := tools.Catalog().ToolNames(); !reflect.DeepEqual(names, []string{"studio.sdk.versions.descriptor"}) {
		t.Fatalf("MCP tools=%v", names)
	}
	request := func(subject string, versionNo int) *http.Request {
		body, _ := json.Marshal(map[string]any{"reportId": "r1", "versionNo": versionNo})
		req := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/versions.descriptor", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if subject != "" {
			req.Header.Set("Authorization", jwt.Bearer(t, subject))
		}
		return req
	}
	scope, err := requestprovider.New(request("viewer", 1))
	if err != nil {
		t.Fatal(err)
	}
	value, err := runtime.ExecuteRoute(ctx, http.MethodPost, "/v1/studio/sdk/versions.descriptor", scope)
	scope.Close()
	if err != nil {
		t.Fatal(err)
	}
	output := value.(*VersionDescriptorOutput)
	if string(output.Component) != `{"route":"/x"}` || string(output.Types) != `{"record":"x"}` || string(output.Resources) != `{"files":[]}` {
		t.Fatalf("descriptor=%+v", output)
	}
	handler := gateway.NewHandler(runtime, nil, "test")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request("viewer", 1))
	if response.Code != http.StatusOK || bytes.Contains(response.Body.Bytes(), []byte("source must not leak")) {
		t.Fatalf("HTTP status=%d body=%s", response.Code, response.Body.String())
	}
	var wire map[string]any
	if err = json.Unmarshal(response.Body.Bytes(), &wire); err != nil || wire["item"] != nil || wire["component"].(map[string]any)["route"] != "/x" || wire["types"].(map[string]any)["record"] != "x" {
		t.Fatalf("HTTP descriptor=%v err=%v body=%s", wire, err, response.Body.String())
	}
	for _, check := range []struct {
		subject string
		version int
		status  int
	}{{"intruder", 1, 404}, {"viewer", 999, 404}, {"", 1, 401}} {
		denied := httptest.NewRecorder()
		handler.ServeHTTP(denied, request(check.subject, check.version))
		if denied.Code != check.status || bytes.Contains(denied.Body.Bytes(), []byte(`"route":"/x"`)) {
			t.Fatalf("%s/%d status=%d body=%s", check.subject, check.version, denied.Code, denied.Body.String())
		}
	}
	document, err := (openapi.Generator{}).Generate(ctx, openapi.Request{Info: openapi3.Info{Title: "Studio SDK", Version: "1"}, Components: []*registry.RegisteredComponent{authEntry, entry}, Routes: []spec.RouteRef{{Method: http.MethodPost, Path: "/v1/studio/sdk/versions.descriptor"}}})
	if err != nil || document.Paths["/v1/studio/sdk/versions.descriptor"].Post == nil {
		t.Fatalf("OpenAPI descriptor route: %v", err)
	}
	tool, ok := tools.Registry().ToolRegistry.Get("studio.sdk.versions.descriptor")
	if !ok {
		t.Fatal("MCP descriptor tool is missing")
	}
	call := func(subject string) ([]byte, error) {
		callContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, subject)})
		result, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall, Params: schema.CallToolRequestParams{Name: "studio.sdk.versions.descriptor", Arguments: map[string]any{"reportId": "r1", "versionNo": 1}}})
		if rpcErr != nil {
			return nil, rpcErr
		}
		if result.IsError != nil && *result.IsError {
			return nil, nil
		}
		return json.Marshal(result.StructuredContent)
	}
	structured, err := call("viewer")
	if err != nil || !bytes.Contains(structured, []byte(`"route":"/x"`)) || bytes.Contains(structured, []byte("source must not leak")) {
		t.Fatalf("MCP descriptor=%s err=%v", structured, err)
	}
	if _, err = db.ExecContext(ctx, "DELETE FROM component_acl WHERE report_id = ? AND subject_id = ?", "r1", "viewer"); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request("viewer", 1))
	if response.Code != http.StatusNotFound {
		t.Fatalf("revoked view grant status=%d body=%s", response.Code, response.Body.String())
	}
}
