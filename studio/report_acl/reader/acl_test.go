package reader

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
	"github.com/viant/datly/gateway/openapi/openapi3"
	"github.com/viant/datly/mcp"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
	dtag "github.com/viant/datly/tag"
	"github.com/viant/mcp-protocol/authorization"
	"github.com/viant/mcp-protocol/schema"
	xresponse "github.com/viant/xdatly/response"
)

func TestACLListSDKDatlyRouteAndMCPContract(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "sdk_acl_list", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{
			"name": "main", "driver": "sqlite", "owner_id": "alice", "status": "active",
			"etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00",
		}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{{
			"owner_id": "alice", "name": "general", "title": "General", "status": "active",
			"etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00",
		}, {
			"owner_id": "bob", "name": "general", "title": "General", "status": "active",
			"etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00",
		}}},
		datatest.Table{Name: "reports", Rows: []datatest.Row{{
			"id": "r1", "slug": "first", "title": "First", "owner_id": "alice",
			"status": "active", "default_connector_name": "main",
			"component_scope": "reports/first", "component_name": "first",
			"etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00",
		}}},
		datatest.Table{Name: "report_acl", Rows: []datatest.Row{{
			"report_id": "r1", "subject_type": "user", "subject_id": "bob",
			"can_view": true, "can_edit": true, "etag": 7,
		}}},
	); err != nil {
		t.Fatal(err)
	}
	holder := reflect.TypeFor[AclComponent]()
	field, ok := holder.FieldByName("Contract")
	if !ok {
		t.Fatal("missing generated SDK component")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatal(err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name,
		PackageName: "reader", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(
		reflect.TypeFor[Input](), reflect.TypeFor[Output]())
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(AclDatlyResourceNamespace, AclDatlyResources); err != nil {
		t.Fatal(err)
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: reflect.TypeFor[Input](), OutputType: reflect.TypeFor[Output](),
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
	entry.Capabilities.Connector = connector
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{entry}, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })
	service, err := mcp.New(mcp.Config{Components: []*registry.RegisteredComponent{entry}, Invoker: runtime, Resources: resources})
	if err != nil {
		t.Fatal(err)
	}
	if names := service.Catalog().ToolNames(); !reflect.DeepEqual(names, []string{"studio.sdk.acl.list"}) {
		t.Fatalf("MCP tools=%v", names)
	}
	tool, ok := service.Registry().ToolRegistry.Get("studio.sdk.acl.list")
	if !ok {
		t.Fatal("SDK ACL MCP tool is missing")
	}
	plan, _ := service.Catalog().Tool("studio.sdk.acl.list")
	if arguments := plan.Arguments(); len(arguments) != 3 || arguments[0].PublicName() != "reportId" ||
		arguments[0].SourceName() != "reportId" || arguments[0].SourceKind() != "body" {
		t.Fatalf("MCP ACL input contract=%+v", arguments)
	}
	callTool := func(subject string) (*schema.CallToolResult, string) {
		t.Helper()
		callContext := ctx
		if subject != "" {
			callContext = context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, subject)})
		}
		result, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall,
			Params: schema.CallToolRequestParams{Name: "studio.sdk.acl.list", Arguments: map[string]any{"reportId": "r1"}}})
		if rpcErr != nil {
			return nil, rpcErr.Message
		}
		return result, ""
	}
	for _, subject := range []string{"alice"} {
		result, message := callTool(subject)
		if message != "" || result == nil || result.IsError != nil && *result.IsError {
			t.Fatalf("MCP %s returned error: %+v %s", subject, result, message)
		}
		body, marshalErr := json.Marshal(result.StructuredContent)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		var wire map[string]any
		if err = json.Unmarshal(body, &wire); err != nil {
			t.Fatal(err)
		}
		items, ok := wire["items"].([]any)
		if !ok || len(items) != 1 {
			t.Fatalf("MCP %s structured output=%s", subject, body)
		}
		row, ok := items[0].(map[string]any)
		if !ok || row["canEdit"] != true || row["etag"] != float64(7) {
			t.Fatalf("MCP %s ACL row=%v", subject, items[0])
		}
	}
	for _, subject := range []string{"bob", "charlie", ""} {
		result, message := callTool(subject)
		if message == "" && result != nil && (result.IsError == nil || !*result.IsError) {
			body, _ := json.Marshal(result.StructuredContent)
			if subject == "" || bytes.Contains(body, []byte(`"subjectId":"bob"`)) {
				t.Fatalf("MCP %s exposed ACL: %+v", subject, result)
			}
		}
	}

	invoke := func(subject string) (*Output, error) {
		request := httptest.NewRequest("POST", "/v1/studio/sdk/acl.list", bytes.NewBufferString(`{"reportId":"r1"}`))
		request.Header.Set("Content-Type", "application/json")
		if subject != "" {
			request.Header.Set("Authorization", jwt.Bearer(t, subject))
		}
		scope, scopeErr := requestprovider.New(request)
		if scopeErr != nil {
			t.Fatal(scopeErr)
		}
		defer scope.Close()
		value, invokeErr := runtime.ExecuteRoute(ctx, "POST", "/v1/studio/sdk/acl.list", scope)
		if invokeErr != nil {
			return nil, invokeErr
		}
		return value.(*Output), nil
	}
	assertDenied := func(subject string) {
		t.Helper()
		_, invokeErr := invoke(subject)
		var forbidden *xresponse.Error
		if !errors.As(invokeErr, &forbidden) || forbidden.Code != http.StatusForbidden {
			t.Fatalf("%s ACL error=%v, want 403", subject, invokeErr)
		}
	}
	for _, subject := range []string{"alice"} {
		output, invokeErr := invoke(subject)
		if invokeErr != nil || output == nil || len(output.Items) != 1 || !output.Items[0].CanEdit ||
			output.Items[0].Etag == nil || *output.Items[0].Etag != 7 {
			t.Fatalf("%s output=%+v err=%v", subject, output, invokeErr)
		}
		body, marshalErr := json.Marshal(output)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		var decoded map[string]any
		if err = json.Unmarshal(body, &decoded); err != nil {
			t.Fatal(err)
		}
		items, ok := decoded["Items"].([]any)
		if !ok || len(items) != 1 {
			t.Fatalf("SDK wire body=%s", body)
		}
	}
	for _, subject := range []string{"bob", "charlie"} {
		assertDenied(subject)
	}
	if _, err = invoke(""); err == nil {
		t.Fatal("missing verified JWT was accepted")
	}

	handler := gateway.NewHandler(runtime, nil, "test")
	request := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/acl.list", bytes.NewBufferString(`{"reportId":"r1"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", jwt.Bearer(t, "alice"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("Datly SDK HTTP status=%d body=%s", response.Code, response.Body.String())
	}
	var wire map[string]any
	if err = json.Unmarshal(response.Body.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	items, ok := wire["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("Datly SDK HTTP body=%s", response.Body.String())
	}
	row, ok := items[0].(map[string]any)
	if !ok || row["canEdit"] != true || row["etag"] != float64(7) {
		t.Fatalf("Datly SDK ACL row=%v", items[0])
	}
	deniedRequest := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/acl.list", bytes.NewBufferString(`{"reportId":"r1"}`))
	deniedRequest.Header.Set("Content-Type", "application/json")
	deniedRequest.Header.Set("Authorization", jwt.Bearer(t, "bob"))
	deniedResponse := httptest.NewRecorder()
	handler.ServeHTTP(deniedResponse, deniedRequest)
	if deniedResponse.Code != http.StatusForbidden || bytes.Contains(deniedResponse.Body.Bytes(), []byte(`"subjectId":"bob"`)) {
		t.Fatalf("delegated editor Datly HTTP status=%d body=%s", deniedResponse.Code, deniedResponse.Body.String())
	}
	documented, err := (gateway.Config{OpenAPI: &gateway.OpenAPIConfig{Info: openapi3.Info{
		Title: "Studio SDK", Version: "1",
	}}}).Build(ctx, gateway.HandlerInput{Runtime: runtime, Components: []*registry.RegisteredComponent{entry}})
	if err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	documented.ServeHTTP(response, httptest.NewRequest(http.MethodGet, gateway.DefaultOpenAPIURI, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("OpenAPI status=%d body=%s", response.Code, response.Body.String())
	}
	var document map[string]any
	if err = json.Unmarshal(response.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	paths, _ := document["paths"].(map[string]any)
	route, _ := paths["/v1/studio/sdk/acl.list"].(map[string]any)
	if _, ok = route["post"]; !ok {
		t.Fatalf("generated OpenAPI lacks SDK POST route: %s", response.Body.String())
	}
	post := route["post"].(map[string]any)
	requestBody, _ := post["requestBody"].(map[string]any)
	if requestBody == nil {
		t.Fatalf("generated OpenAPI lacks SDK request body: %s", response.Body.String())
	}

	if _, err = db.ExecContext(ctx, "UPDATE report_acl SET can_edit = FALSE WHERE report_id = ? AND subject_id = ?", "r1", "bob"); err != nil {
		t.Fatal(err)
	}
	assertDenied("bob")
	if viewerTool, message := callTool("bob"); message == "" && viewerTool != nil && (viewerTool.IsError == nil || !*viewerTool.IsError) {
		body, marshalErr := json.Marshal(viewerTool.StructuredContent)
		if marshalErr != nil || bytes.Contains(body, []byte(`"subjectId":"bob"`)) {
			t.Fatalf("viewer MCP access=%s err=%v", body, marshalErr)
		}
	}
	if _, err = db.ExecContext(ctx, "UPDATE reports SET owner_id = ? WHERE id = ?", "bob", "r1"); err != nil {
		t.Fatal(err)
	}
	assertDenied("alice")
	revokedTool, message := callTool("alice")
	if message == "" && revokedTool != nil && (revokedTool.IsError == nil || !*revokedTool.IsError) {
		body, marshalErr := json.Marshal(revokedTool.StructuredContent)
		if marshalErr != nil || bytes.Contains(body, []byte(`"subjectId":"bob"`)) {
			t.Fatalf("former owner MCP access=%s err=%v", body, marshalErr)
		}
	}
	newOwner, err := invoke("bob")
	if err != nil || newOwner == nil || len(newOwner.Items) != 1 {
		t.Fatalf("new owner HTTP access=%+v err=%v", newOwner, err)
	}
}
