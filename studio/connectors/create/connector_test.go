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

func TestConnectorCreateSDKDatlyHTTPMCPAndOpenAPI(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "sdk_connector_create", "studio")
	jwt := datatest.NewJWTFixture(t)
	holder := reflect.TypeFor[ConnectorComponent]()
	field, ok := holder.FieldByName("Contract")
	if !ok {
		t.Fatal("connector create component is missing")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatalf("component metadata: %v", err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name,
		PackageName: "create", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(
		reflect.TypeFor[ConnectorCreateInput](), reflect.TypeFor[ConnectorCreateOutput]())
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(ConnectorDatlyResourceNamespace, ConnectorDatlyResources); err != nil {
		t.Fatal(err)
	}
	authEntry := datatest.AuthRegistration(t, db, jwt.Factory, resources)
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: reflect.TypeFor[ConnectorCreateInput](), OutputType: reflect.TypeFor[ConnectorCreateOutput](),
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
	writer, err := writerhandler.New(component, reflect.TypeFor[ConnectorCreateInput](), reflect.TypeFor[ConnectorCreateOutput](), "post")
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
	if names := tools.Catalog().ToolNames(); !reflect.DeepEqual(names, []string{"studio.sdk.connectors.create"}) {
		t.Fatalf("MCP tools=%v", names)
	}
	handler := gateway.NewHandler(runtime, nil, "test")
	request := func(subject, body string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/connectors.create", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		if subject != "" {
			req.Header.Set("Authorization", jwt.Bearer(t, subject))
		}
		return req
	}
	const privateDSN, privateSecret = "root:private@tcp(localhost:3306)/app", "vault://private/mysql"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request("alice", `{"name":"warehouse","driver":"mysql","dsnTemplate":"`+privateDSN+`","secretRef":"`+privateSecret+`","description":"Operations","ownerId":"alice","options":{"pool":3},"status":"active","etag":99}`))
	if response.Code != http.StatusOK {
		t.Fatalf("create status=%d body=%s", response.Code, response.Body.String())
	}
	if bytes.Contains(response.Body.Bytes(), []byte(privateDSN)) || bytes.Contains(response.Body.Bytes(), []byte(privateSecret)) {
		t.Fatalf("connector create response leaked connection material: %s", response.Body.String())
	}
	var wire map[string]any
	if err = json.Unmarshal(response.Body.Bytes(), &wire); err != nil || wire["name"] != "warehouse" || wire["driver"] != "mysql" || wire["ownerId"] != "alice" || wire["status"] != "draft" || wire["etag"] != float64(1) || wire["dsnConfigured"] != true || wire["secretConfigured"] != true || wire["data"] != nil {
		t.Fatalf("created connector=%v err=%v body=%s", wire, err, response.Body.String())
	}
	var owner, status, dsn, secret, options string
	var etag int
	if err = db.QueryRowContext(ctx, "SELECT owner_id,status,etag,dsn_template,secret_ref,options_json FROM connectors WHERE name=?", "warehouse").Scan(&owner, &status, &etag, &dsn, &secret, &options); err != nil || owner != "alice" || status != "draft" || etag != 1 || dsn != privateDSN || secret != privateSecret || options != `{"pool":3}` {
		t.Fatalf("stored connector owner=%q status=%q etag=%d dsn=%q secret=%q options=%q err=%v", owner, status, etag, dsn, secret, options, err)
	}
	for _, format := range []string{"csv", "xml", "xlsx", "tabular"} {
		blocked := request("alice", `{"name":"blocked-`+format+`","driver":"mysql","dsnTemplate":"`+privateDSN+`","secretRef":"`+privateSecret+`"}`)
		blocked.URL.RawQuery = "_format=" + format
		denied := httptest.NewRecorder()
		handler.ServeHTTP(denied, blocked)
		if denied.Code != http.StatusNotAcceptable || bytes.Contains(denied.Body.Bytes(), []byte(privateDSN)) || bytes.Contains(denied.Body.Bytes(), []byte(privateSecret)) {
			t.Fatalf("custom JSON output format %s status=%d body=%s", format, denied.Code, denied.Body.String())
		}
		var count int
		if err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM connectors WHERE name=?", "blocked-"+format).Scan(&count); err != nil || count != 0 {
			t.Fatalf("rejected format %s executed connector mutation: count=%d err=%v", format, count, err)
		}
	}
	for _, check := range []struct {
		subject, body string
		status        int
	}{{"alice", `{"name":"other","driver":"sqlite","ownerId":"bob"}`, 403}, {"alice", `{"name":"invalid"}`, 400}, {"", `{"name":"anonymous","driver":"sqlite"}`, 401}} {
		denied := httptest.NewRecorder()
		handler.ServeHTTP(denied, request(check.subject, check.body))
		if denied.Code != check.status {
			t.Fatalf("subject=%q body=%s status=%d response=%s", check.subject, check.body, denied.Code, denied.Body.String())
		}
	}
	duplicate := httptest.NewRecorder()
	handler.ServeHTTP(duplicate, request("alice", `{"name":"warehouse","driver":"mysql"}`))
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate status=%d body=%s", duplicate.Code, duplicate.Body.String())
	}
	document, err := (openapi.Generator{}).Generate(ctx, openapi.Request{Info: openapi3.Info{Title: "Studio SDK", Version: "1"}, Components: []*registry.RegisteredComponent{authEntry, entry}, Routes: []spec.RouteRef{{Method: "POST", Path: "/v1/studio/sdk/connectors.create"}}})
	if err != nil || document.Paths["/v1/studio/sdk/connectors.create"].Post == nil {
		t.Fatalf("OpenAPI create route: %v", err)
	}
	operation := document.Paths["/v1/studio/sdk/connectors.create"].Post
	requestSchema := operation.RequestBody.Content["application/json"].Schema
	responseSchema := operation.Responses["200"].Content["application/json"].Schema
	if requestSchema == nil || responseSchema == nil || !strings.HasPrefix(requestSchema.Ref, "#/components/schemas/") || !strings.HasPrefix(responseSchema.Ref, "#/components/schemas/") {
		t.Fatalf("OpenAPI create request=%+v response=%+v", requestSchema, responseSchema)
	}
	requestBody := document.Components.Schemas[strings.TrimPrefix(requestSchema.Ref, "#/components/schemas/")]
	responseBody := document.Components.Schemas[strings.TrimPrefix(responseSchema.Ref, "#/components/schemas/")]
	if requestBody == nil || requestBody.Properties["dsnTemplate"] == nil || requestBody.Properties["secretRef"] == nil || requestBody.Properties["status"] != nil || requestBody.Properties["etag"] != nil || responseBody == nil || responseBody.Properties["dsnTemplate"] != nil || responseBody.Properties["secretRef"] != nil || responseBody.Properties["dsnConfigured"] == nil || responseBody.Properties["secretConfigured"] == nil {
		t.Fatalf("OpenAPI create request=%+v response=%+v", requestBody, responseBody)
	}
	tool, ok := tools.Registry().ToolRegistry.Get("studio.sdk.connectors.create")
	if !ok {
		t.Fatal("MCP create tool is missing")
	}
	callContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, "bob")})
	result, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall, Params: schema.CallToolRequestParams{Name: "studio.sdk.connectors.create", Arguments: map[string]any{"name": "analytics", "driver": "sqlite", "dsnTemplate": privateDSN, "secretRef": privateSecret}}})
	if rpcErr != nil || result == nil || result.IsError != nil && *result.IsError {
		t.Fatalf("MCP create result=%+v err=%v", result, rpcErr)
	}
	structured, err := json.Marshal(result.StructuredContent)
	if err != nil || !bytes.Contains(structured, []byte(`"name":"analytics"`)) || bytes.Contains(structured, []byte(privateDSN)) || bytes.Contains(structured, []byte(privateSecret)) {
		t.Fatalf("MCP create response leaked connection material: %s err=%v", structured, err)
	}
	var bobOwner string
	if err = db.QueryRowContext(ctx, "SELECT owner_id FROM connectors WHERE name=?", "analytics").Scan(&bobOwner); err != nil || bobOwner != "bob" {
		t.Fatalf("MCP-created owner=%q err=%v", bobOwner, err)
	}
}
