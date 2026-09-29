package access

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	reports "github.com/viant/datly-studio/studio/reports/get"
	"github.com/viant/datly/bootstrap"
	gateway "github.com/viant/datly/gateway/http"
	"github.com/viant/datly/mcp"
	druntime "github.com/viant/datly/runtime"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
	dtag "github.com/viant/datly/tag"
	"github.com/viant/mcp-protocol/authorization"
	"github.com/viant/mcp-protocol/schema"
)

func TestNativePermissionCatalogNamespaceHTTPAndMCP(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "permission_namespace", "studio")
	jwt := datatest.NewJWTFixture(t)
	if _, err := db.Exec(`INSERT INTO connectors(name,driver,owner_id,status,etag,created_at,updated_at) VALUES('main','sqlite','owner','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"alpha", "beta"} {
		id := namespaceaccess.ID("owner", name)
		if _, err := db.Exec(`INSERT INTO namespaces(namespace_id,owner_id,name,title,status,etag,created_at,updated_at) VALUES(?,'owner',?,?,'active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, id, name, name); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO components(id,slug,title,owner_id,status,namespace,namespace_id,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES(?,?,?,'owner','active',?,?,'main',?,?,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, name, name, name, name, id, name, name); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO report_versions(report_id,version_no,state,authoring_mode,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,created_by,created_at) VALUES(?,1,'draft','dql','{}','1','hash','{}','pending','1','1','owner',CURRENT_TIMESTAMP)`, name); err != nil {
			t.Fatal(err)
		}
	}
	for _, key := range []string{"STUDIO_ACCESS_ISSUER", "STUDIO_ACCESS_AUDIENCE", "STUDIO_ACCESS_PUBLIC_KEY_FILE", "STUDIO_ACCESS_USER_INFO_URL"} {
		t.Setenv(key, "")
	}
	resources := resource.New()
	if err := resources.Register(reports.ReportDatlyResourceNamespace, reports.ReportDatlyResources); err != nil {
		t.Fatal(err)
	}
	connector := &dsql.SQLComponent{DB: db}
	if err := connector.RegisterConnector("studio", db); err != nil {
		t.Fatal(err)
	}
	compile := func(holder, input, output reflect.Type, handler rhandler.TypedHandler) *registry.RegisteredComponent {
		field, _ := holder.FieldByName("Contract")
		metadata, present, err := dtag.ParseComponent(field.Tag)
		if err != nil || !present {
			t.Fatalf("metadata: %v", err)
		}
		component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackageName: holder.Name(), PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(input, output)
		if err != nil {
			t.Fatal(err)
		}
		artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: input, OutputType: output, Resources: resources, Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwt.Factory, Handler: handler, HandlerOwnedOutput: handler != nil})
		if err != nil {
			t.Fatal(err)
		}
		registration := registry.RegisteredComponent{}
		registration.Capabilities.Connector = connector
		if handler == nil {
			registration.Reader, err = artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: connector})
			if err != nil {
				t.Fatal(err)
			}
		}
		entry, err := artifact.Registration(registration)
		if err != nil {
			t.Fatal(err)
		}
		return entry
	}
	handler, err := (ListComponent{}).DatlyHandler("NewList")()
	if err != nil {
		t.Fatal(err)
	}
	entries := []*registry.RegisteredComponent{datatest.AuthRegistration(t, db, jwt.Factory, resources),
		compile(reflect.TypeFor[reports.ReportComponent](), reflect.TypeFor[reports.ReportGetInput](), reflect.TypeFor[reports.ReportGetOutput](), nil),
		compile(reflect.TypeFor[ListComponent](), reflect.TypeFor[ListInput](), reflect.TypeFor[ListOutput](), handler)}
	runtime, err := druntime.NewRuntime(entries, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })
	bearer := jwt.BearerWithClaims(t, jwtv5.MapClaims{"sub": "owner", "iss": "test", "exp": time.Now().Add(time.Hour).Unix()})
	httpHandler := gateway.NewHandler(runtime, nil, "test")
	check := func(raw []byte, want string) {
		var page struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &page); err != nil {
			t.Fatal(err)
		}
		if len(page.Items) != 1 || page.Items[0].ID != want {
			t.Fatalf("catalog escaped namespace: %s", raw)
		}
	}
	for _, name := range []string{"alpha", "beta"} {
		req := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/access.list", bytes.NewBufferString(`{"limit":1}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", bearer)
		req.Header.Set("X-Studio-Namespace", namespaceaccess.ID("owner", name))
		res := httptest.NewRecorder()
		httpHandler.ServeHTTP(res, req)
		if res.Code != 200 {
			t.Fatalf("HTTP %d: %s", res.Code, res.Body.String())
		}
		check(res.Body.Bytes(), name)
	}
	service, err := mcp.New(mcp.Config{Components: entries, Invoker: runtime, Resources: resources})
	if err != nil {
		t.Fatal(err)
	}
	tool, ok := service.Registry().ToolRegistry.Get("studio.sdk.access.list")
	if !ok {
		t.Fatal("permissions MCP tool missing")
	}
	callCtx := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: bearer})
	result, mcpErr := tool.Handler(callCtx, &schema.CallToolRequest{Method: schema.MethodToolsCall, Params: schema.CallToolRequestParams{Name: "studio.sdk.access.list", Arguments: map[string]any{"namespaceId": namespaceaccess.ID("owner", "beta"), "limit": 1}}})
	if mcpErr != nil || result == nil || (result.IsError != nil && *result.IsError) {
		t.Fatalf("MCP: %+v %v", result, mcpErr)
	}
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	check(raw, "beta")
}
