package access

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/viant/authz"
	sharedapi "github.com/viant/authz/component/api"
	policyreader "github.com/viant/authz/component/policy/reader"
	policystore "github.com/viant/authz/component/store/sql"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	reports "github.com/viant/datly-studio/studio/reports/get"
	policycatalog "github.com/viant/datly-studio/studio/resource_policy/catalog"
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
	factoryProvider := nativeFactorySubprocess(t, "TestNativePermissionCatalogNamespaceHTTPAndMCP")
	if factoryProvider == nil {
		return
	}
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
	store := &policystore.Store{DB: db}
	t.Cleanup(func() { _ = store.Close(ctx) })
	for _, name := range []string{"alpha", "beta"} {
		_, err := store.Provision(ctx, authz.Document{Resource: authz.Resource{Kind: "component", ID: name, Version: "1", Tenant: "tenant"}, Policies: map[string]authz.Policy{
			"viewAccess":   {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: "owner"}},
			"manageAccess": {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: "owner"}},
		}}, "fixture")
		if err != nil {
			t.Fatal(err)
		}
	}
	keyPath := filepath.Join(t.TempDir(), "public.pem")
	if err := os.WriteFile(keyPath, jwt.PublicKeyPEM(t), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STUDIO_ACCESS_ISSUER", "test")
	t.Setenv("STUDIO_ACCESS_AUDIENCE", "studio")
	t.Setenv("STUDIO_ACCESS_PUBLIC_KEY_FILE", keyPath)
	t.Setenv("STUDIO_ACCESS_USER_INFO_URL", "")
	resources := resource.New()
	if err := resources.Register(reports.ReportDatlyResourceNamespace, reports.ReportDatlyResources); err != nil {
		t.Fatal(err)
	}
	if err := resources.Register(policycatalog.Namespace, policycatalog.Resources); err != nil {
		t.Fatal(err)
	}
	if err := resources.Register(policyreader.PolicyDatlyResourceNamespace, policyreader.PolicyDatlyResources); err != nil {
		t.Fatal(err)
	}
	connector := &dsql.SQLComponent{DB: db}
	if err := connector.RegisterConnector("studio", db); err != nil {
		t.Fatal(err)
	}
	if err := connector.RegisterConnector("authz", db); err != nil {
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
		compile(reflect.TypeFor[policyreader.PolicyComponent](), reflect.TypeFor[policyreader.Input](), reflect.TypeFor[policyreader.Output](), nil),
		compile(reflect.TypeFor[policycatalog.Component](), reflect.TypeFor[policycatalog.Input](), reflect.TypeFor[policycatalog.Output](), nil),
		compile(reflect.TypeFor[reports.ReportComponent](), reflect.TypeFor[reports.ReportGetInput](), reflect.TypeFor[reports.ReportGetOutput](), nil),
		compile(reflect.TypeFor[ListComponent](), reflect.TypeFor[ListInput](), reflect.TypeFor[ListOutput](), handler)}
	for _, item := range []struct {
		holder, input, output reflect.Type
		factory               func() (rhandler.TypedHandler, error)
	}{
		{reflect.TypeFor[GetComponent](), reflect.TypeFor[ResourceInput](), reflect.TypeFor[DocumentOutput](), (GetComponent{}).DatlyHandler("NewGet")},
		{reflect.TypeFor[PolicyGetComponent](), reflect.TypeFor[PolicyInput](), reflect.TypeFor[sharedapi.PolicyOutput](), (PolicyGetComponent{}).DatlyHandler("NewPolicyGet")},
		{reflect.TypeFor[PolicyContextComponent](), reflect.TypeFor[PolicyInput](), reflect.TypeFor[sharedapi.PolicyContextOutput](), (PolicyContextComponent{}).DatlyHandler("NewPolicyContext")},
		{reflect.TypeFor[PolicyReplaceComponent](), reflect.TypeFor[PolicyWriteInput](), reflect.TypeFor[sharedapi.PolicyOutput](), (PolicyReplaceComponent{}).DatlyHandler("NewPolicyReplace")},
		{reflect.TypeFor[AuthorizationComponent](), reflect.TypeFor[AuthorizationInput](), reflect.TypeFor[sharedapi.DecisionOutput](), (AuthorizationComponent{}).DatlyHandler("NewAuthorizationCheck")},
		{reflect.TypeFor[ContextComponent](), reflect.TypeFor[ResourceInput](), reflect.TypeFor[ContextOutput](), (ContextComponent{}).DatlyHandler("NewContext")},
		{reflect.TypeFor[ReplaceComponent](), reflect.TypeFor[ReplaceInput](), reflect.TypeFor[DocumentOutput](), (ReplaceComponent{}).DatlyHandler("NewReplace")},
	} {
		handler, err := item.factory()
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, compile(item.holder, item.input, item.output, handler))
	}
	runtime, err := druntime.NewRuntime(entries, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })
	bearer := jwt.BearerWithClaims(t, jwtv5.MapClaims{"sub": "owner", "iss": "test", "aud": "studio", "tenant": "tenant", "exp": time.Now().Add(time.Hour).Unix()})
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
	for _, operation := range []string{"get", "context"} {
		req := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/access."+operation, bytes.NewBufferString(`{"kind":"component","id":"alpha","version":"1","tenant":"tenant"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", bearer)
		req.Header.Set("X-Studio-Namespace", namespaceaccess.ID("owner", "alpha"))
		res := httptest.NewRecorder()
		httpHandler.ServeHTTP(res, req)
		if res.Code != 200 {
			t.Fatalf("local %s HTTP %d: %s", operation, res.Code, res.Body.String())
		}
	}
	for _, operation := range []string{"get", "context", "replace"} {
		resource := map[string]any{"kind": "component", "id": "beta", "version": "1", "tenant": "tenant"}
		var body any = resource
		if operation == "replace" {
			body = map[string]any{"resource": resource, "revision": 1, "policies": map[string]any{}}
		}
		payload, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/access."+operation, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", bearer)
		req.Header.Set("X-Studio-Namespace", namespaceaccess.ID("owner", "alpha"))
		res := httptest.NewRecorder()
		httpHandler.ServeHTTP(res, req)
		if res.Code != 403 {
			t.Fatalf("foreign %s HTTP %d: %s", operation, res.Code, res.Body.String())
		}
		arguments := map[string]any{}
		for key, value := range body.(map[string]any) {
			arguments[key] = value
		}
		arguments["namespaceId"] = namespaceaccess.ID("owner", "alpha")
		tool, ok := service.Registry().ToolRegistry.Get("studio.sdk.access." + operation)
		if !ok {
			t.Fatal("management MCP tool missing")
		}
		result, mcpErr := tool.Handler(callCtx, &schema.CallToolRequest{Method: schema.MethodToolsCall, Params: schema.CallToolRequestParams{Name: "studio.sdk.access." + operation, Arguments: arguments}})
		if mcpErr == nil && result != nil && (result.IsError == nil || !*result.IsError) {
			t.Fatalf("foreign %s MCP accepted", operation)
		}
	}
	// The canonical shared wire contract uses wrapped inputs and outputs over both transports.
	resource := map[string]any{"kind": "component", "id": "alpha", "version": "1", "tenant": "tenant"}
	for _, operation := range []struct {
		name, field string
		input       map[string]any
	}{
		{"policies.get", "document", map[string]any{"resource": resource}},
		{"policies.context", "context", map[string]any{"resource": resource}},
		{"authorization.check", "decision", map[string]any{"resource": resource, "action": "viewAccess"}},
	} {
		payload, _ := json.Marshal(operation.input)
		req := httptest.NewRequest(http.MethodPost, "/v1/authz/sdk/"+operation.name, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", bearer)
		req.Header.Set("X-Studio-Namespace", namespaceaccess.ID("owner", "alpha"))
		res := httptest.NewRecorder()
		httpHandler.ServeHTTP(res, req)
		if res.Code != 200 {
			t.Fatalf("shared %s HTTP %d: %s", operation.name, res.Code, res.Body.String())
		}
		var wire map[string]any
		if err := json.Unmarshal(res.Body.Bytes(), &wire); err != nil || wire[operation.field] == nil {
			t.Fatalf("shared wrapped output: %s (%v)", res.Body.String(), err)
		}
		if operation.field == "context" && wire["context"].(map[string]any)["canManage"] != true {
			t.Fatalf("shared management authority: %s", res.Body.String())
		}
		tool, ok := service.Registry().ToolRegistry.Get("authz.sdk." + operation.name)
		if !ok {
			t.Fatal("shared MCP tool missing")
		}
		arguments := map[string]any{"namespaceId": namespaceaccess.ID("owner", "alpha")}
		for key, value := range operation.input {
			arguments[key] = value
		}
		result, err := tool.Handler(callCtx, &schema.CallToolRequest{Method: schema.MethodToolsCall, Params: schema.CallToolRequestParams{Name: "authz.sdk." + operation.name, Arguments: arguments}})
		if err != nil || result == nil || result.IsError != nil && *result.IsError {
			t.Fatalf("shared %s MCP: %+v %v", operation.name, result, err)
		}
		raw, _ := json.Marshal(result.StructuredContent)
		var mcpWire map[string]any
		if err := json.Unmarshal(raw, &mcpWire); err != nil || mcpWire[operation.field] == nil {
			t.Fatalf("shared MCP wrapped output %s", raw)
		}
		req = httptest.NewRequest(http.MethodPost, "/v1/authz/sdk/"+operation.name, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", bearer)
		req.Header.Set("X-Studio-Namespace", namespaceaccess.ID("owner", "beta"))
		res = httptest.NewRecorder()
		httpHandler.ServeHTTP(res, req)
		if res.Code != 403 {
			t.Fatalf("shared foreign %s HTTP %d: %s", operation.name, res.Code, res.Body.String())
		}
		arguments["namespaceId"] = namespaceaccess.ID("owner", "beta")
		result, err = tool.Handler(callCtx, &schema.CallToolRequest{Method: schema.MethodToolsCall, Params: schema.CallToolRequestParams{Name: "authz.sdk." + operation.name, Arguments: arguments}})
		if err == nil && result != nil && (result.IsError == nil || !*result.IsError) {
			t.Fatalf("shared foreign %s MCP accepted", operation.name)
		}
	}
	if factoryProvider.calls == 0 {
		t.Fatal("native HTTP handlers did not resolve the registered startup factory")
	}
	factoryProvider.subject = "changed-authority"
	deniedByChangedAuthority := httptest.NewRecorder()
	changedRequest := httptest.NewRequest(http.MethodPost, "/v1/authz/sdk/policies.get", bytes.NewBufferString(`{"resource":{"kind":"component","id":"alpha","version":"1","tenant":"tenant"}}`))
	changedRequest.Header.Set("Content-Type", "application/json")
	changedRequest.Header.Set("Authorization", bearer)
	changedRequest.Header.Set("X-Studio-Namespace", namespaceaccess.ID("owner", "alpha"))
	httpHandler.ServeHTTP(deniedByChangedAuthority, changedRequest)
	if deniedByChangedAuthority.Code != http.StatusForbidden && deniedByChangedAuthority.Code != http.StatusUnauthorized {
		t.Fatalf("native policy handler accepted changed authority: HTTP %d: %s", deniedByChangedAuthority.Code, deniedByChangedAuthority.Body.String())
	}
	factoryProvider.subject = "owner"
	// CAS rejects an outdated revision before writing, with the shared safe error.
	stale := map[string]any{"document": map[string]any{"resource": resource, "revision": 0, "policies": map[string]any{"viewAccess": map[string]any{"mode": "protected", "rule": map[string]any{"kind": "subject", "value": "owner"}}}}}
	payload, _ := json.Marshal(stale)
	req := httptest.NewRequest(http.MethodPost, "/v1/authz/sdk/policies.replace", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", bearer)
	res := httptest.NewRecorder()
	httpHandler.ServeHTTP(res, req)
	if res.Code != 409 {
		t.Fatalf("shared stale replace %d: %s", res.Code, res.Body.String())
	}

	denyDocument, err := store.Get(ctx, authz.Resource{Kind: "component", ID: "alpha", Version: "1", Tenant: "tenant"})
	if err != nil {
		t.Fatal(err)
	}
	denyDocument.Policies["manageAccess"] = authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: "another"}}
	denyDocument, err = store.Replace(ctx, denyDocument, denyDocument.Revision, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	callHTTP := func(operation string, input any, want int) {
		t.Helper()
		body, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/v1/authz/sdk/"+operation, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", bearer)
		res := httptest.NewRecorder()
		httpHandler.ServeHTTP(res, req)
		if res.Code != want {
			t.Fatalf("%s want %d got %d: %s", operation, want, res.Code, res.Body.String())
		}
		if strings.Contains(res.Body.String(), "policies_json") || strings.Contains(res.Body.String(), "invalid character") {
			t.Fatalf("private storage error leaked: %s", res.Body.String())
		}
	}
	callHTTP("policies.replace", map[string]any{"document": denyDocument}, 403)
	denyDocument.Policies["viewAccess"] = authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: "another"}}
	denyDocument, err = store.Replace(ctx, denyDocument, denyDocument.Revision, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	callHTTP("policies.get", map[string]any{"resource": resource}, 403)
	callHTTP("policies.context", map[string]any{"resource": resource}, 403)
	// A broken trusted backing store is unavailable, never an allow or a detailed SQL error.
	if _, err := db.Exec("UPDATE resource_policy_revisions SET policies_json='broken-json' WHERE resource_id='alpha'"); err != nil {
		t.Fatal(err)
	}
	callHTTP("policies.get", map[string]any{"resource": resource}, 503)
	callHTTP("policies.context", map[string]any{"resource": resource}, 503)
	callHTTP("authorization.check", map[string]any{"resource": resource, "action": "viewAccess"}, 503)

}
