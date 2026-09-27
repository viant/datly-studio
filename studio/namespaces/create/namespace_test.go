package create

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/bindly/locator"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly/bootstrap"
	gateway "github.com/viant/datly/gateway/http"
	"github.com/viant/datly/gateway/openapi"
	"github.com/viant/datly/gateway/openapi/openapi3"
	"github.com/viant/datly/mcp"
	druntime "github.com/viant/datly/runtime"
	writerhandler "github.com/viant/datly/runtime/handler/writer"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/sql/dml"
	viewprovider "github.com/viant/datly/sql/reader/provider"
	dtag "github.com/viant/datly/tag"
	"github.com/viant/mcp-protocol/authorization"
	"github.com/viant/mcp-protocol/schema"
)

func TestNamespaceCreateSDKDatlyHTTPMCPAndOpenAPI(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "sdk_namespace_create", "studio")
	jwt := datatest.NewJWTFixture(t)
	holder := reflect.TypeFor[NamespaceComponent]()
	field, ok := holder.FieldByName("Contract")
	if !ok {
		t.Fatal("namespace create component is missing")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatalf("component metadata: %v", err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name,
		PackageName: "create", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(
		reflect.TypeFor[NamespaceCreateInput](), reflect.TypeFor[NamespaceCreateOutput]())
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(NamespaceDatlyResourceNamespace, NamespaceDatlyResources); err != nil {
		t.Fatal(err)
	}
	authEntry := datatest.AuthRegistration(t, db, jwt.Factory, resources)
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: reflect.TypeFor[NamespaceCreateInput](), OutputType: reflect.TypeFor[NamespaceCreateOutput](),
		Resources: resources, Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwt.Factory})
	if err != nil {
		t.Fatal(err)
	}
	connector := &dsql.SQLComponent{DB: db}
	if err = connector.RegisterConnector("studio", db); err != nil {
		t.Fatal(err)
	}
	var providers []locator.Provider
	if len(artifact.ViewDependencies) > 0 {
		views, providerErr := viewprovider.New(viewprovider.Config{Dependencies: artifact.ViewDependencies, Input: artifact.Input, SQL: connector})
		if providerErr != nil {
			t.Fatal(providerErr)
		}
		providers = append(providers, views)
	}
	writer, err := writerhandler.New(component, reflect.TypeFor[NamespaceCreateInput](), reflect.TypeFor[NamespaceCreateOutput](), "post")
	if err != nil {
		t.Fatal(err)
	}
	registration := registry.RegisteredComponent{Handler: writer, Providers: providers, DataSource: dml.Source{DB: db}}
	registration.Capabilities.Connector = connector
	entry, err := artifact.Registration(registration)
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
	if names := tools.Catalog().ToolNames(); !reflect.DeepEqual(names, []string{"studio.sdk.namespaces.create"}) {
		t.Fatalf("MCP tools=%v", names)
	}
	handler := gateway.NewHandler(runtime, nil, "test")
	request := func(subject, body string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/namespaces.create", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		if subject != "" {
			req.Header.Set("Authorization", jwt.Bearer(t, subject))
		}
		return req
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request("alice", `{"name":"finance.ops","title":"Finance","ownerId":"alice","status":"archived","etag":99,"createdAt":"2000-01-01T00:00:00Z"}`))
	if response.Code != http.StatusOK {
		t.Fatalf("create status=%d body=%s", response.Code, response.Body.String())
	}
	var wire map[string]any
	if err = json.Unmarshal(response.Body.Bytes(), &wire); err != nil || wire["ownerId"] != "alice" || wire["name"] != "finance.ops" || wire["title"] != "Finance" || wire["status"] != "active" || wire["etag"] != float64(1) || wire["data"] != nil {
		t.Fatalf("created namespace=%v err=%v body=%s", wire, err, response.Body.String())
	}
	var owner, status string
	var etag int
	if err = db.QueryRowContext(ctx, "SELECT owner_id,status,etag FROM namespaces WHERE name=?", "finance.ops").Scan(&owner, &status, &etag); err != nil || owner != "alice" || status != "active" || etag != 1 {
		t.Fatalf("stored namespace owner=%q status=%q etag=%d err=%v", owner, status, etag, err)
	}
	for _, check := range []struct {
		subject, body string
		status        int
	}{{"alice", `{"name":"other","title":"Other","ownerId":"bob"}`, 403}, {"alice", `{"name":"Bad.Name","title":"Bad"}`, 400}, {"", `{"name":"anonymous","title":"Anonymous"}`, 401}} {
		denied := httptest.NewRecorder()
		handler.ServeHTTP(denied, request(check.subject, check.body))
		if denied.Code != check.status {
			t.Fatalf("subject=%q body=%s status=%d response=%s", check.subject, check.body, denied.Code, denied.Body.String())
		}
	}
	duplicate := httptest.NewRecorder()
	handler.ServeHTTP(duplicate, request("alice", `{"name":"finance.ops","title":"Again"}`))
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate status=%d body=%s", duplicate.Code, duplicate.Body.String())
	}
	document, err := (openapi.Generator{}).Generate(ctx, openapi.Request{Info: openapi3.Info{Title: "Studio SDK", Version: "1"}, Components: []*registry.RegisteredComponent{authEntry, entry}, Routes: []spec.RouteRef{{Method: "POST", Path: "/v1/studio/sdk/namespaces.create"}}})
	if err != nil || document.Paths["/v1/studio/sdk/namespaces.create"].Post == nil {
		t.Fatalf("OpenAPI create route: %v", err)
	}
	requestSchema := document.Paths["/v1/studio/sdk/namespaces.create"].Post.RequestBody.Content["application/json"].Schema
	if requestSchema == nil || !strings.HasPrefix(requestSchema.Ref, "#/components/schemas/") {
		t.Fatalf("OpenAPI create request schema=%+v", requestSchema)
	}
	requestBody := document.Components.Schemas[strings.TrimPrefix(requestSchema.Ref, "#/components/schemas/")]
	if requestBody == nil || requestBody.Properties["name"] == nil || requestBody.Properties["title"] == nil ||
		requestBody.Properties["status"] != nil || requestBody.Properties["etag"] != nil || requestBody.Properties["createdAt"] != nil {
		t.Fatalf("OpenAPI create request DTO=%+v", requestBody)
	}
	responseSchema := document.Paths["/v1/studio/sdk/namespaces.create"].Post.Responses["200"].Content["application/json"].Schema
	if responseSchema == nil || !strings.HasPrefix(responseSchema.Ref, "#/components/schemas/") {
		t.Fatalf("OpenAPI create response schema=%+v", responseSchema)
	}
	resolved := document.Components.Schemas[strings.TrimPrefix(responseSchema.Ref, "#/components/schemas/")]
	if resolved == nil || resolved.Properties["ownerId"] == nil || resolved.Properties["etag"] == nil || resolved.Properties["createdAt"] == nil {
		t.Fatalf("OpenAPI create DTO=%+v", resolved)
	}
	for _, field := range []string{"ownerId", "name", "title", "status", "etag", "createdAt", "updatedAt"} {
		found := false
		for _, required := range resolved.Required {
			found = found || required == field
		}
		if !found {
			t.Errorf("OpenAPI create response does not require %s", field)
		}
	}
	tool, ok := tools.Registry().ToolRegistry.Get("studio.sdk.namespaces.create")
	if !ok {
		t.Fatal("MCP create tool is missing")
	}
	callContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, "bob")})
	result, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall, Params: schema.CallToolRequestParams{Name: "studio.sdk.namespaces.create", Arguments: map[string]any{"name": "research", "title": "Research"}}})
	if rpcErr != nil || result == nil || result.IsError != nil && *result.IsError {
		t.Fatalf("MCP create result=%+v err=%v", result, rpcErr)
	}
	structured, err := json.Marshal(result.StructuredContent)
	if err != nil || !bytes.Contains(structured, []byte(`"name":"research"`)) || bytes.Contains(structured, []byte(`"data":`)) {
		t.Fatalf("MCP create DTO=%s err=%v", structured, err)
	}
	var bobOwner string
	if err = db.QueryRowContext(ctx, "SELECT owner_id FROM namespaces WHERE name=?", "research").Scan(&bobOwner); err != nil || bobOwner != "bob" {
		t.Fatalf("MCP-created owner=%q err=%v", bobOwner, err)
	}
}
