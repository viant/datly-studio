package list

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
)

func TestVersionListSDKDatlyHTTPMCPAndOpenAPI(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "sdk_version_list", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{"name": "main", "driver": "sqlite", "owner_id": "alice", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{{"owner_id": "alice", "name": "general", "title": "General", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "components", Rows: []datatest.Row{{"id": "r1", "slug": "first", "title": "First", "owner_id": "alice", "status": "active", "default_connector_name": "main", "namespace": "general", "component_scope": "reports/first", "component_name": "first", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "component_acl", Rows: []datatest.Row{
			{"report_id": "r1", "subject_type": "user", "subject_id": "bob", "can_view": true},
			{"report_id": "r1", "subject_type": "user", "subject_id": "carol", "can_view": true, "can_edit": true, "can_use_dql": true},
		}},
		datatest.Table{Name: "component_versions", Rows: []datatest.Row{
			{"report_id": "r1", "version_no": 1, "state": "published", "authoring_mode": "dql", "authored_sql": "SELECT 1", "authored_dql": "secret one", "generated_dql": "generated one", "component_spec_json": "{}", "spec_format_version": "1", "spec_hash": "hash-1", "type_manifest_json": "{}", "compile_status": "valid", "datly_version": "v1", "compiler_version": "v1", "source_revision": 1, "created_by": "alice", "created_at": "2026-09-17 09:00:00"},
			{"report_id": "r1", "version_no": 2, "state": "draft", "authoring_mode": "dql", "authored_sql": "SELECT 2", "authored_dql": "secret two", "generated_dql": "generated two", "component_spec_json": "{}", "spec_format_version": "1", "spec_hash": "hash-2", "type_manifest_json": "{}", "compile_status": "valid", "datly_version": "v1", "compiler_version": "v1", "source_revision": 2, "created_by": "alice", "created_at": "2026-09-18 09:00:00"},
			{"report_id": "r1", "version_no": 3, "state": "draft", "authoring_mode": "structured", "authored_sql": "SELECT 3", "authored_dql": "secret three", "generated_dql": "generated three", "component_spec_json": "{}", "spec_format_version": "1", "spec_hash": "hash-3", "type_manifest_json": "{}", "compile_status": "invalid", "compile_diagnostics_json": `[{"severity":"error","message":"secret three"}]`, "datly_version": "v1", "compiler_version": "v1", "source_revision": 3, "created_by": "bob", "created_at": "2026-09-19 09:00:00"},
		}},
	); err != nil {
		t.Fatal(err)
	}
	holder := reflect.TypeFor[VersionComponent]()
	field, ok := holder.FieldByName("Contract")
	if !ok {
		t.Fatal("version-list component is missing")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatalf("component metadata: %v", err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackageName: "list", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(reflect.TypeFor[VersionListInput](), reflect.TypeFor[VersionListOutput]())
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(VersionDatlyResourceNamespace, VersionDatlyResources); err != nil {
		t.Fatal(err)
	}
	authEntry := datatest.AuthRegistration(t, db, jwt.Factory, resources)
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeFor[VersionListInput](), OutputType: reflect.TypeFor[VersionListOutput](), Resources: resources, Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwt.Factory})
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
	if names := tools.Catalog().ToolNames(); !reflect.DeepEqual(names, []string{"studio.sdk.versions.list"}) {
		t.Fatalf("MCP tools=%v", names)
	}
	request := func(subject, body string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/versions.list", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		if subject != "" {
			req.Header.Set("Authorization", jwt.Bearer(t, subject))
		}
		return req
	}
	invoke := func(subject, body string) (*VersionListOutput, error) {
		scope, scopeErr := requestprovider.New(request(subject, body))
		if scopeErr != nil {
			t.Fatal(scopeErr)
		}
		defer scope.Close()
		value, invokeErr := runtime.ExecuteRoute(ctx, http.MethodPost, "/v1/studio/sdk/versions.list", scope)
		if invokeErr != nil {
			return nil, invokeErr
		}
		return value.(*VersionListOutput), nil
	}
	page, err := invoke("bob", `{"reportId":"r1","input":{}}`)
	if err != nil || page.PageLimit != 50 || page.PageOffset != 0 || len(page.Items) != 3 || page.Items[0].VersionNo != 3 || page.Items[2].VersionNo != 1 || page.Items[0].AuthoredDql != nil || page.Items[0].GeneratedDql != nil || bytes.Contains(page.Items[0].CompileDiagnosticsJson, []byte("secret three")) {
		t.Fatalf("viewer page=%+v err=%v", page, err)
	}
	page, err = invoke("alice", `{"reportId":"r1","input":{"limit":1,"offset":1}}`)
	if err != nil || page.PageLimit != 1 || page.PageOffset != 1 || len(page.Items) != 1 || page.Items[0].VersionNo != 2 || page.Items[0].AuthoredDql == nil {
		t.Fatalf("owner page=%+v err=%v", page, err)
	}
	page, err = invoke("carol", `{"reportId":"r1","input":{"state":"draft","authoringMode":"dql","compileStatus":"valid","createdBy":"alice","limit":5000,"orderBy":"version_no ASC","fields":["state"]}}`)
	if err != nil || page.PageLimit != 500 || len(page.Items) != 1 || page.Items[0].VersionNo != 2 || page.Items[0].AuthoredDql == nil {
		t.Fatalf("filtered DQL-author page=%+v err=%v", page, err)
	}
	page, err = invoke("bob", `{"reportId":"r1","input":{"offset":-9}}`)
	if err != nil || page.PageOffset != 0 {
		t.Fatalf("negative offset page=%+v err=%v", page, err)
	}
	page, err = invoke("mallory", `{"reportId":"r1","input":{}}`)
	if err == nil && page != nil && len(page.Items) > 0 {
		t.Fatalf("unauthorized list returned items: %+v", page)
	}
	handler := gateway.NewHandler(runtime, nil, "test")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request("bob", `{"reportId":"r1","input":{"limit":1}}`))
	if response.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d body=%s", response.Code, response.Body.String())
	}
	var wire map[string]any
	if err = json.Unmarshal(response.Body.Bytes(), &wire); err != nil || wire["limit"] != float64(1) || wire["offset"] != float64(0) || bytes.Contains(response.Body.Bytes(), []byte("secret")) {
		t.Fatalf("HTTP viewer page=%v err=%v body=%s", wire, err, response.Body.String())
	}
	document, err := (openapi.Generator{}).Generate(ctx, openapi.Request{Info: openapi3.Info{Title: "Studio SDK", Version: "1"}, Components: []*registry.RegisteredComponent{authEntry, entry}, Routes: []spec.RouteRef{{Method: http.MethodPost, Path: "/v1/studio/sdk/versions.list"}}})
	if err != nil || document.Paths["/v1/studio/sdk/versions.list"].Post == nil {
		t.Fatalf("OpenAPI list route: %v", err)
	}
	tool, ok := tools.Registry().ToolRegistry.Get("studio.sdk.versions.list")
	if !ok {
		t.Fatal("MCP list tool is missing")
	}

	generalID := namespaceaccess.ID("alice", "general")
	otherID := namespaceaccess.ID("alice", "other")
	if _, err := db.ExecContext(ctx, `UPDATE namespaces SET namespace_id=? WHERE owner_id='alice' AND name='general'`, generalID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO namespaces(namespace_id,owner_id,name,title,status,etag,created_at,updated_at) VALUES(?,'alice','other','Other','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, otherID); err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct {
		subject, id string
		count       int
	}{{"alice", generalID, 1}, {"alice", otherID, 0}, {"bob", generalID, 0}} {
		scopedRequest := request(check.subject, `{"reportId":"r1","input":{"limit":1,"offset":1}}`)
		scopedRequest.Header.Set("X-Studio-Namespace", check.id)
		scopedResponse := httptest.NewRecorder()
		handler.ServeHTTP(scopedResponse, scopedRequest)
		var selected struct {
			Items []map[string]any `json:"items"`
		}
		if err := json.Unmarshal(scopedResponse.Body.Bytes(), &selected); scopedResponse.Code != 200 || err != nil || len(selected.Items) != check.count {
			t.Fatalf("version namespace HTTP subject=%s count=%d body=%s err=%v", check.subject, check.count, scopedResponse.Body.String(), err)
		}
		scopedContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, check.subject)})
		scopedResult, scopedErr := tool.Handler(scopedContext, &schema.CallToolRequest{Method: schema.MethodToolsCall, Params: schema.CallToolRequestParams{Name: "studio.sdk.versions.list", Arguments: map[string]any{"reportId": "r1", "namespaceId": check.id, "input": map[string]any{"limit": 1, "offset": 1}}}})
		if scopedErr != nil || scopedResult == nil || scopedResult.IsError != nil && *scopedResult.IsError {
			t.Fatalf("version namespace MCP: %+v err=%v", scopedResult, scopedErr)
		}
		payload, err := json.Marshal(scopedResult.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(payload, &selected); err != nil || len(selected.Items) != check.count {
			t.Fatalf("version namespace MCP subject=%s count=%d payload=%s err=%v", check.subject, check.count, payload, err)
		}
	}
	callContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, "bob")})
	result, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall, Params: schema.CallToolRequestParams{Name: "studio.sdk.versions.list", Arguments: map[string]any{"reportId": "r1", "input": map[string]any{"limit": 1}}}})
	if rpcErr != nil || result == nil || result.IsError != nil && *result.IsError {
		t.Fatalf("MCP list result=%+v err=%v", result, rpcErr)
	}
	structured, err := json.Marshal(result.StructuredContent)
	if err != nil || bytes.Contains(structured, []byte("secret")) {
		t.Fatalf("MCP viewer source leaked: %s err=%v", structured, err)
	}
	if _, err = db.ExecContext(ctx, "DELETE FROM component_acl WHERE report_id = ? AND subject_id = ?", "r1", "bob"); err != nil {
		t.Fatal(err)
	}
	page, err = invoke("bob", `{"reportId":"r1","input":{}}`)
	if err == nil && page != nil && len(page.Items) > 0 {
		t.Fatalf("revoked list returned items: %+v", page)
	}
}
