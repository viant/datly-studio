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

func TestPublicationEventsListSDKDatlyHTTPMCPAndOpenAPI(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "sdk_publication_events_list", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{"name": "main", "driver": "sqlite", "owner_id": "alice", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{{"owner_id": "alice", "name": "general", "title": "General", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}, {"owner_id": "carol", "name": "general", "title": "General", "status": "active", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "reports", Rows: []datatest.Row{{"id": "r1", "slug": "first", "title": "First", "owner_id": "alice", "status": "active", "default_connector_name": "main", "namespace": "general", "component_scope": "reports/first", "component_name": "first", "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"}}},
		datatest.Table{Name: "report_acl", Rows: []datatest.Row{{"report_id": "r1", "subject_type": "user", "subject_id": "bob", "can_view": true, "can_publish": true}}},
		datatest.Table{Name: "report_publication_events", Rows: []datatest.Row{
			{"event_id": "e1", "report_id": "r1", "owner_id": "alice", "operation": "publish", "status": "succeeded", "requested_by": "alice", "occurred_at": "2026-09-17 10:00:00"},
			{"event_id": "e2", "report_id": "r1", "owner_id": "alice", "operation": "rollback", "status": "failed", "requested_by": "alice", "reason": "retry", "failure_code": "timeout", "failure_message": "failed", "occurred_at": "2026-09-17 11:00:00"},
			{"event_id": "e3", "report_id": "r1", "owner_id": "alice", "operation": "publish", "status": "succeeded", "requested_by": "alice", "occurred_at": "2026-09-17 12:00:00"},
		}},
	); err != nil {
		t.Fatal(err)
	}
	holder := reflect.TypeFor[EventComponent]()
	field, ok := holder.FieldByName("Contract")
	if !ok {
		t.Fatal("generated publication-event list component is missing")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatalf("component metadata: %v", err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name,
		PackageName: "list", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(
		reflect.TypeFor[PublicationEventsListInput](), reflect.TypeFor[PublicationEventsListOutput]())
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(EventDatlyResourceNamespace, EventDatlyResources); err != nil {
		t.Fatal(err)
	}
	authEntry := datatest.AuthRegistration(t, db, jwt.Factory, resources)
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: reflect.TypeFor[PublicationEventsListInput](), OutputType: reflect.TypeFor[PublicationEventsListOutput](),
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
	tools, err := mcp.New(mcp.Config{Components: []*registry.RegisteredComponent{authEntry, entry}, Invoker: runtime, Resources: resources})
	if err != nil {
		t.Fatal(err)
	}
	if names := tools.Catalog().ToolNames(); !reflect.DeepEqual(names, []string{"studio.sdk.publications.events.list"}) {
		t.Fatalf("MCP tools=%v", names)
	}
	invoke := func(subject, body string) (*PublicationEventsListOutput, error) {
		request := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/publications.events.list", bytes.NewBufferString(body))
		request.Header.Set("Content-Type", "application/json")
		if subject != "" {
			request.Header.Set("Authorization", jwt.Bearer(t, subject))
		}
		scope, scopeErr := requestprovider.New(request)
		if scopeErr != nil {
			t.Fatal(scopeErr)
		}
		defer scope.Close()
		value, invokeErr := runtime.ExecuteRoute(ctx, http.MethodPost, "/v1/studio/sdk/publications.events.list", scope)
		if invokeErr != nil {
			return nil, invokeErr
		}
		return value.(*PublicationEventsListOutput), nil
	}
	page, err := invoke("alice", `{"reportId":"r1","input":{}}`)
	if err != nil || page.PageLimit != 20 || page.PageOffset != 0 || len(page.Items) != 3 || page.Items[0].EventId != "e3" || page.Items[2].EventId != "e1" {
		t.Fatalf("default event page=%+v err=%v", page, err)
	}
	page, err = invoke("alice", `{"reportId":"r1"}`)
	if err != nil || page.PageLimit != 20 || len(page.Items) != 3 {
		t.Fatalf("omitted nested input page=%+v err=%v", page, err)
	}
	page, err = invoke("alice", `{"reportId":"r1","input":{"limit":101}}`)
	if err != nil || page.PageLimit != 100 || len(page.Items) != 3 {
		t.Fatalf("capped event page=%+v err=%v", page, err)
	}
	page, err = invoke("alice", `{"reportId":"r1","input":{"operation":" rollback ","status":" failed ","limit":1,"offset":0}}`)
	if err != nil || len(page.Items) != 1 || page.Items[0].EventId != "e2" || page.Items[0].FailureCode == nil || *page.Items[0].FailureCode != "timeout" {
		t.Fatalf("filtered event page=%+v err=%v", page, err)
	}
	page, err = invoke("alice", `{"reportId":"r1","input":{"limit":1,"offset":1}}`)
	if err != nil || page.PageLimit != 1 || page.PageOffset != 1 || len(page.Items) != 1 || page.Items[0].EventId != "e2" {
		t.Fatalf("paged event history=%+v err=%v", page, err)
	}
	for _, subject := range []string{"bob", "dave"} {
		denied, invokeErr := invoke(subject, `{"reportId":"r1","input":{}}`)
		if invokeErr == nil && denied != nil && len(denied.Items) != 0 {
			t.Fatalf("%s received owner history: %+v", subject, denied)
		}
	}
	for _, body := range []string{`{"reportId":"r1","input":{"operation":"delete"}}`, `{"reportId":"r1","input":{"offset":-1}}`} {
		if _, err = invoke("alice", body); err == nil {
			t.Fatalf("invalid event list input %s was accepted", body)
		}
	}
	handler := gateway.NewHandler(runtime, nil, "test")
	request := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/publications.events.list", bytes.NewBufferString(`{"reportId":"r1","input":{"limit":1}}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", jwt.Bearer(t, "alice"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d body=%s", response.Code, response.Body.String())
	}
	var wire map[string]any
	if err = json.Unmarshal(response.Body.Bytes(), &wire); err != nil || wire["limit"] != float64(1) || wire["offset"] != float64(0) {
		t.Fatalf("HTTP event page=%v err=%v body=%s", wire, err, response.Body.String())
	}
	items, ok := wire["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("HTTP items=%v", wire["items"])
	}
	item := items[0].(map[string]any)
	if item["eventId"] != "e3" || item["reportId"] != "r1" || item["ownerId"] != "alice" ||
		item["operation"] != "publish" || item["status"] != "succeeded" || item["requestedBy"] != "alice" ||
		item["occurredAt"] == nil || item["reason"] != nil || item["failureCode"] != nil {
		t.Fatalf("HTTP publication event=%v", item)
	}
	document, err := (openapi.Generator{}).Generate(ctx, openapi.Request{
		Info:       openapi3.Info{Title: "Studio SDK", Version: "1"},
		Components: []*registry.RegisteredComponent{authEntry, entry},
		Routes:     []spec.RouteRef{{Method: http.MethodPost, Path: "/v1/studio/sdk/publications.events.list"}},
	})
	if err != nil || document.Paths["/v1/studio/sdk/publications.events.list"].Post == nil {
		t.Fatalf("OpenAPI event route missing: %v", err)
	}
	tool, ok := tools.Registry().ToolRegistry.Get("studio.sdk.publications.events.list")
	if !ok {
		t.Fatal("MCP event-list tool is missing")
	}
	callContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, "alice")})
	result, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall,
		Params: schema.CallToolRequestParams{Name: "studio.sdk.publications.events.list", Arguments: map[string]any{"reportId": "r1", "input": map[string]any{"limit": 1}}}})
	if rpcErr != nil || result == nil || result.IsError != nil && *result.IsError {
		t.Fatalf("MCP event page=%+v err=%v", result, rpcErr)
	}
	structured, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var mcpWire map[string]any
	if err = json.Unmarshal(structured, &mcpWire); err != nil || mcpWire["limit"] != float64(1) {
		t.Fatalf("MCP event page=%s err=%v", structured, err)
	}
	mcpItems, ok := mcpWire["items"].([]any)
	if !ok || len(mcpItems) != 1 || mcpItems[0].(map[string]any)["eventId"] != "e3" {
		t.Fatalf("MCP events=%v", mcpWire["items"])
	}
	if _, err = db.ExecContext(ctx, `UPDATE reports SET owner_id='carol' WHERE id='r1'`); err != nil {
		t.Fatal(err)
	}
	former, formerErr := invoke("alice", `{"reportId":"r1","input":{}}`)
	if formerErr == nil && former != nil && len(former.Items) != 0 {
		t.Fatalf("former owner event history=%+v", former)
	}
	current, currentErr := invoke("carol", `{"reportId":"r1","input":{}}`)
	if currentErr != nil || len(current.Items) != 3 {
		t.Fatalf("current owner event history=%+v err=%v", current, currentErr)
	}
}
