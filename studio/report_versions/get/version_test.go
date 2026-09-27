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

func TestVersionGetScopesMetadataAndRedactsDQLAcrossProtocols(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "sdk_version_get", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{"name": "main", "driver": "sqlite", "owner_id": "alice", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{{"owner_id": "alice", "name": "general", "title": "General", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "components", Rows: []datatest.Row{{"id": "r1", "slug": "first", "title": "First", "owner_id": "alice", "status": "active", "default_connector_name": "main", "namespace": "general", "component_scope": "reports/first", "component_name": "first", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "report_acl", Rows: []datatest.Row{
			{"report_id": "r1", "subject_type": "user", "subject_id": "bob", "can_view": true},
			{"report_id": "r1", "subject_type": "user", "subject_id": "carol", "can_view": true, "can_edit": true, "can_use_dql": true},
		}},
		datatest.Table{Name: "report_versions", Rows: []datatest.Row{{
			"report_id": "r1", "version_no": 2, "state": "draft", "authoring_mode": "dql",
			"authored_sql": "SELECT 1", "authored_dql": "secret authored DQL", "generated_dql": "secret generated DQL",
			"component_spec_json": "{}", "spec_format_version": "1", "spec_hash": "hash", "type_manifest_json": "{}",
			"compile_status": "valid", "compile_diagnostics_json": `[{"severity":"error","message":"secret authored DQL"}]`,
			"datly_version": "v1", "compiler_version": "v1", "source_revision": 3,
			"created_by": "alice", "created_at": "2026-09-17 09:00:00",
		}}},
	); err != nil {
		t.Fatal(err)
	}
	holder := reflect.TypeFor[VersionComponent]()
	field, ok := holder.FieldByName("Contract")
	if !ok {
		t.Fatal("generated version-get component is missing")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatalf("component metadata: %v", err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name,
		PackageName: "get", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(
		reflect.TypeFor[VersionGetInput](), reflect.TypeFor[VersionGetOutput]())
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(VersionDatlyResourceNamespace, VersionDatlyResources); err != nil {
		t.Fatal(err)
	}
	authEntry := datatest.AuthRegistration(t, db, jwt.Factory, resources)
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: reflect.TypeFor[VersionGetInput](), OutputType: reflect.TypeFor[VersionGetOutput](),
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
	if names := toolService.Catalog().ToolNames(); !reflect.DeepEqual(names, []string{"studio.sdk.versions.get"}) {
		t.Fatalf("MCP tools=%v", names)
	}
	request := func(subject string, versionNo int) *http.Request {
		body, _ := json.Marshal(map[string]any{"reportId": "r1", "versionNo": versionNo})
		req := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/versions.get", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if subject != "" {
			req.Header.Set("Authorization", jwt.Bearer(t, subject))
		}
		return req
	}
	invoke := func(subject string, versionNo int) (*VersionGetOutput, error) {
		scope, scopeErr := requestprovider.New(request(subject, versionNo))
		if scopeErr != nil {
			t.Fatal(scopeErr)
		}
		defer scope.Close()
		value, invokeErr := runtime.ExecuteRoute(ctx, http.MethodPost, "/v1/studio/sdk/versions.get", scope)
		if invokeErr != nil {
			return nil, invokeErr
		}
		return value.(*VersionGetOutput), nil
	}
	for _, check := range []struct {
		subject string
		wantDQL bool
	}{{"alice", true}, {"bob", false}, {"carol", true}} {
		output, invokeErr := invoke(check.subject, 2)
		if invokeErr != nil || output == nil || output.Item == nil || output.ResponseReportId != "r1" || output.ResponseVersionNo != 2 || output.SourceRevision != 3 {
			t.Fatalf("%s version=%+v err=%v", check.subject, output, invokeErr)
		}
		if (output.AuthoredDql != "") != check.wantDQL || (output.GeneratedDql != "") != check.wantDQL {
			t.Fatalf("%s DQL exposure authored=%q generated=%q", check.subject, output.AuthoredDql, output.GeneratedDql)
		}
		if bytes.Contains(output.CompileDiagnostics, []byte("secret authored DQL")) != check.wantDQL {
			t.Fatalf("%s diagnostics exposure=%s", check.subject, output.CompileDiagnostics)
		}
	}
	for _, check := range []struct {
		subject   string
		versionNo int
	}{{"mallory", 2}, {"alice", 99}} {
		_, invokeErr := invoke(check.subject, check.versionNo)
		var notFound *xresponse.Error
		if !errors.As(invokeErr, &notFound) || notFound.Code != http.StatusNotFound {
			t.Fatalf("%s/%d error=%v, want 404", check.subject, check.versionNo, invokeErr)
		}
	}
	if _, err = invoke("", 2); err == nil {
		t.Fatal("missing JWT was accepted")
	}
	handler := gateway.NewHandler(runtime, nil, "test")
	for _, check := range []struct {
		subject string
		wantDQL bool
	}{{"alice", true}, {"bob", false}} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request(check.subject, 2))
		if response.Code != http.StatusOK {
			t.Fatalf("HTTP %s status=%d body=%s", check.subject, response.Code, response.Body.String())
		}
		var wire map[string]any
		if err = json.Unmarshal(response.Body.Bytes(), &wire); err != nil || wire["reportId"] != "r1" || wire["versionNo"] != float64(2) || wire["item"] != nil {
			t.Fatalf("HTTP %s version=%v err=%v", check.subject, wire, err)
		}
		if (wire["authoredDql"] != nil) != check.wantDQL || (wire["generatedDql"] != nil) != check.wantDQL ||
			bytes.Contains(response.Body.Bytes(), []byte("secret authored DQL")) != check.wantDQL {
			t.Fatalf("HTTP %s source=%v", check.subject, wire)
		}
	}
	for _, check := range []struct {
		subject string
		version int
		status  int
	}{{"alice", 99, http.StatusNotFound}, {"mallory", 2, http.StatusNotFound}, {"", 2, http.StatusUnauthorized}} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request(check.subject, check.version))
		if response.Code != check.status || bytes.Contains(response.Body.Bytes(), []byte("secret authored DQL")) {
			t.Fatalf("HTTP %s/%d status=%d body=%s", check.subject, check.version, response.Code, response.Body.String())
		}
	}
	document, err := (openapi.Generator{}).Generate(ctx, openapi.Request{
		Info:       openapi3.Info{Title: "Studio SDK", Version: "1"},
		Components: []*registry.RegisteredComponent{authEntry, entry},
		Routes:     []spec.RouteRef{{Method: http.MethodPost, Path: "/v1/studio/sdk/versions.get"}},
	})
	if err != nil || document.Paths["/v1/studio/sdk/versions.get"].Post == nil {
		t.Fatalf("OpenAPI versions.get route missing: %v", err)
	}
	tool, ok := toolService.Registry().ToolRegistry.Get("studio.sdk.versions.get")
	if !ok {
		t.Fatal("MCP version-get tool is missing")
	}
	for _, check := range []struct {
		subject string
		wantDQL bool
	}{{"alice", true}, {"bob", false}} {
		callContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, check.subject)})
		result, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall,
			Params: schema.CallToolRequestParams{Name: "studio.sdk.versions.get", Arguments: map[string]any{"reportId": "r1", "versionNo": 2}}})
		if rpcErr != nil || result == nil || result.IsError != nil && *result.IsError {
			t.Fatalf("MCP %s version=%+v err=%v", check.subject, result, rpcErr)
		}
		structured, marshalErr := json.Marshal(result.StructuredContent)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		var wire map[string]any
		if err = json.Unmarshal(structured, &wire); err != nil || wire["reportId"] != "r1" || wire["versionNo"] != float64(2) || (wire["authoredDql"] != nil) != check.wantDQL {
			t.Fatalf("MCP %s version=%s err=%v", check.subject, structured, err)
		}
	}
	if _, err = db.ExecContext(ctx, "DELETE FROM report_acl WHERE report_id = ? AND subject_id = ?", "r1", "bob"); err != nil {
		t.Fatal(err)
	}
	_, err = invoke("bob", 2)
	var notFound *xresponse.Error
	if !errors.As(err, &notFound) || notFound.Code != http.StatusNotFound {
		t.Fatalf("revoked reader error=%v, want 404", err)
	}
	callContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, "bob")})
	revoked, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall,
		Params: schema.CallToolRequestParams{Name: "studio.sdk.versions.get", Arguments: map[string]any{"reportId": "r1", "versionNo": 2}}})
	if rpcErr == nil && revoked != nil && (revoked.IsError == nil || !*revoked.IsError) {
		t.Fatalf("revoked MCP reader returned data: %+v", revoked)
	}
}
