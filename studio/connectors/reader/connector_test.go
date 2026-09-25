package reader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
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

func TestConnectorListSDKDatlyContract(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "sdk_connector_list", "studio")
	jwt := datatest.NewJWTFixture(t)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{
			{"name": "alpha", "driver": "sqlite", "dsn_template": "sqlite://private-token", "description": "Primary analytics", "owner_id": "owner-a", "status": "active", "options_json": `{}`, "etag": 1, "created_at": "2026-09-17 10:00:00", "updated_at": "2026-09-17 10:00:00"},
			{"name": "beta", "driver": "mysql", "secret_ref": "secret://private-token", "description": "Archive", "owner_id": "owner-b", "status": "disabled", "etag": 2, "created_at": "2026-09-17 11:00:00", "updated_at": "2026-09-17 11:00:00"},
			{"name": "gamma", "driver": "mysql", "description": "Gamma analytics", "owner_id": "owner-a", "status": "active", "etag": 3, "created_at": "2026-09-17 12:00:00", "updated_at": "2026-09-17 12:00:00"},
			{"name": "removed", "driver": "mysql", "owner_id": "owner-a", "status": "deleted", "etag": 4, "created_at": "2026-09-17 13:00:00", "updated_at": "2026-09-17 13:00:00", "deleted_at": "2026-09-17 13:30:00"},
		}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{
			{"owner_id": "owner-a", "name": "general", "title": "General", "status": "active", "etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"},
			{"owner_id": "owner-b", "name": "general", "title": "General", "status": "active", "etag": 1, "created_at": "2026-09-17 09:00:00", "updated_at": "2026-09-17 09:00:00"},
		}},
		datatest.Table{Name: "reports", Rows: []datatest.Row{
			{"id": "r-alpha", "slug": "alpha", "title": "Alpha", "owner_id": "owner-a", "status": "active", "default_connector_name": "alpha", "component_scope": "reports", "component_name": "alpha", "created_at": "2026-09-17 10:00:00", "updated_at": "2026-09-17 10:00:00"},
			{"id": "r-beta", "slug": "beta", "title": "Beta", "owner_id": "owner-b", "status": "active", "default_connector_name": "beta", "component_scope": "reports", "component_name": "beta", "created_at": "2026-09-17 10:00:00", "updated_at": "2026-09-17 10:00:00"},
			{"id": "r-gamma", "slug": "gamma", "title": "Gamma", "owner_id": "owner-a", "status": "active", "default_connector_name": "gamma", "component_scope": "reports", "component_name": "gamma", "created_at": "2026-09-17 10:00:00", "updated_at": "2026-09-17 10:00:00"},
		}},
		datatest.Table{Name: "report_acl", Rows: []datatest.Row{
			{"report_id": "r-alpha", "subject_type": "user", "subject_id": "viewer", "can_view": true},
			{"report_id": "r-beta", "subject_type": "user", "subject_id": "viewer", "can_view": true},
			{"report_id": "r-gamma", "subject_type": "user", "subject_id": "viewer", "can_view": true},
		}},
	); err != nil {
		t.Fatal(err)
	}
	holder := reflect.TypeFor[ConnectorComponent]()
	field, ok := holder.FieldByName("Contract")
	if !ok {
		t.Fatal("missing generated connector-list component")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatalf("component metadata: %v", err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name,
		PackageName: "reader", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(
		reflect.TypeFor[Input](), reflect.TypeFor[Output]())
	if err != nil {
		t.Fatal(err)
	}
	resources := resource.New()
	if err = resources.Register(ConnectorDatlyResourceNamespace, ConnectorDatlyResources); err != nil {
		t.Fatal(err)
	}
	authEntry := datatest.AuthRegistration(t, db, jwt.Factory, resources)
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
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{authEntry, entry}, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })
	service, err := mcp.New(mcp.Config{Components: []*registry.RegisteredComponent{authEntry, entry}, Invoker: runtime, Resources: resources})
	if err != nil {
		t.Fatal(err)
	}
	if names := service.Catalog().ToolNames(); !reflect.DeepEqual(names, []string{"studio.sdk.connectors.list"}) {
		t.Fatalf("MCP tools=%v", names)
	}
	invokeAs := func(subject string, input map[string]any) (*Output, error) {
		payload, marshalErr := json.Marshal(input)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		request := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/connectors.list", bytes.NewReader(payload))
		request.Header.Set("Content-Type", "application/json")
		if subject != "" {
			request.Header.Set("Authorization", jwt.Bearer(t, subject))
		}
		scope, scopeErr := requestprovider.New(request)
		if scopeErr != nil {
			t.Fatal(scopeErr)
		}
		defer scope.Close()
		value, invokeErr := runtime.ExecuteRoute(ctx, http.MethodPost, "/v1/studio/sdk/connectors.list", scope)
		if invokeErr != nil {
			return nil, invokeErr
		}
		return value.(*Output), nil
	}
	names := func(output *Output) []string {
		if output == nil {
			return nil
		}
		result := make([]string, 0, len(output.Items))
		for _, item := range output.Items {
			if item != nil {
				result = append(result, item.Name)
			}
		}
		return result
	}
	for _, check := range []struct {
		name  string
		input map[string]any
		want  []string
	}{
		{"default order", map[string]any{}, []string{"gamma", "beta", "alpha"}},
		{"search description", map[string]any{"query": "analytics"}, []string{"gamma", "alpha"}},
		{"search driver case insensitive", map[string]any{"query": "MYSQL"}, []string{"gamma", "beta"}},
		{"status", map[string]any{"status": "active"}, []string{"gamma", "alpha"}},
		{"owner", map[string]any{"ownerId": "owner-a"}, []string{"gamma", "alpha"}},
		{"driver", map[string]any{"driver": "mysql"}, []string{"gamma", "beta"}},
		{"combined", map[string]any{"status": "active", "ownerId": "owner-a", "driver": "mysql"}, []string{"gamma"}},
		{"empty filter", map[string]any{"status": ""}, []string{"gamma", "beta", "alpha"}},
		{"pagination", map[string]any{"limit": 1, "offset": 1}, []string{"beta"}},
		{"ignored SDK selectors", map[string]any{"fields": []string{"name"}, "orderBy": "name ASC"}, []string{"gamma", "beta", "alpha"}},
	} {
		t.Run(check.name, func(t *testing.T) {
			output, invokeErr := invokeAs("viewer", check.input)
			if invokeErr != nil || !reflect.DeepEqual(names(output), check.want) {
				t.Fatalf("input=%v connectors=%v err=%v", check.input, names(output), invokeErr)
			}
		})
	}
	for _, check := range []struct {
		input  map[string]any
		limit  int
		offset int
	}{{map[string]any{}, 50, 0}, {map[string]any{"limit": 999}, 500, 0}, {map[string]any{"limit": 1, "offset": -5}, 1, 0}} {
		output, invokeErr := invokeAs("viewer", check.input)
		if invokeErr != nil || output.PageLimit != check.limit || output.PageOffset != check.offset {
			t.Fatalf("input=%v page=%+v err=%v", check.input, output, invokeErr)
		}
	}
	owned, err := invokeAs("owner-a", map[string]any{})
	if err != nil || !reflect.DeepEqual(names(owned), []string{"gamma", "alpha"}) {
		t.Fatalf("owner-a connectors=%v err=%v", names(owned), err)
	}
	other, err := invokeAs("owner-b", map[string]any{})
	if err != nil || !reflect.DeepEqual(names(other), []string{"beta"}) {
		t.Fatalf("owner-b connectors=%v err=%v", names(other), err)
	}
	if _, err = invokeAs("", map[string]any{}); err == nil {
		t.Fatal("missing JWT was accepted")
	}
	handler := gateway.NewHandler(runtime, nil, "test")
	request := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/connectors.list", bytes.NewBufferString(`{"limit":1,"offset":1}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", jwt.Bearer(t, "viewer"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("HTTP status=%d body=%s", response.Code, response.Body.String())
	}
	var wire struct {
		Items  []map[string]any `json:"items"`
		Limit  int              `json:"limit"`
		Offset int              `json:"offset"`
	}
	if err = json.Unmarshal(response.Body.Bytes(), &wire); err != nil || wire.Limit != 1 || wire.Offset != 1 ||
		len(wire.Items) != 1 || wire.Items[0]["name"] != "beta" || wire.Items[0]["secretConfigured"] != true {
		t.Fatalf("HTTP page=%+v err=%v body=%s", wire, err, response.Body.String())
	}
	if bytes.Contains(response.Body.Bytes(), []byte("private-token")) || bytes.Contains(response.Body.Bytes(), []byte("dsnTemplate")) || bytes.Contains(response.Body.Bytes(), []byte("secretRef")) {
		t.Fatalf("HTTP connector leaked secret material: %s", response.Body.String())
	}
	response = httptest.NewRecorder()
	alphaRequest := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/connectors.list", bytes.NewBufferString(`{"query":"alpha"}`))
	alphaRequest.Header.Set("Content-Type", "application/json")
	alphaRequest.Header.Set("Authorization", jwt.Bearer(t, "viewer"))
	handler.ServeHTTP(response, alphaRequest)
	if response.Code != http.StatusOK {
		t.Fatalf("alpha HTTP status=%d body=%s", response.Code, response.Body.String())
	}
	var alpha struct {
		Items []map[string]any `json:"items"`
	}
	if err = json.Unmarshal(response.Body.Bytes(), &alpha); err != nil || len(alpha.Items) != 1 || alpha.Items[0]["dsnConfigured"] != true || alpha.Items[0]["options"] == nil {
		t.Fatalf("alpha HTTP page=%+v err=%v body=%s", alpha, err, response.Body.String())
	}
	document, err := (openapi.Generator{}).Generate(ctx, openapi.Request{
		Info:       openapi3.Info{Title: "Studio SDK", Version: "1"},
		Components: []*registry.RegisteredComponent{authEntry, entry},
		Routes:     []spec.RouteRef{{Method: http.MethodPost, Path: "/v1/studio/sdk/connectors.list"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if document.Paths["/v1/studio/sdk/connectors.list"].Post == nil {
		t.Fatal("OpenAPI lacks connectors.list POST route")
	}
	openAPIWire, err := json.Marshal(document)
	if err != nil || bytes.Contains(openAPIWire, []byte(`"dsnTemplate"`)) || bytes.Contains(openAPIWire, []byte(`"secretRef"`)) || bytes.Contains(openAPIWire, []byte(`"subject"`)) {
		t.Fatalf("OpenAPI exposes secrets or trusted scope: %s err=%v", openAPIWire, err)
	}
	tool, ok := service.Registry().ToolRegistry.Get("studio.sdk.connectors.list")
	if !ok {
		t.Fatal("MCP connector-list tool is missing")
	}
	callContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, "viewer")})
	result, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall,
		Params: schema.CallToolRequestParams{Name: "studio.sdk.connectors.list", Arguments: map[string]any{"limit": 1, "offset": 1}}})
	if rpcErr != nil || result == nil || result.IsError != nil && *result.IsError {
		t.Fatalf("MCP page=%+v err=%v", result, rpcErr)
	}
	structured, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var mcpWire map[string]any
	if err = json.Unmarshal(structured, &mcpWire); err != nil || mcpWire["limit"] != float64(1) || mcpWire["offset"] != float64(1) {
		t.Fatalf("MCP page=%s err=%v", structured, err)
	}
	if bytes.Contains(structured, []byte("private-token")) || bytes.Contains(structured, []byte("dsnTemplate")) {
		t.Fatalf("MCP connector leaked secret material: %s", structured)
	}
	if _, err = db.ExecContext(ctx, "UPDATE reports SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?", "r-beta"); err != nil {
		t.Fatal(err)
	}
	revoked, err := invokeAs("viewer", map[string]any{})
	if err != nil || !reflect.DeepEqual(names(revoked), []string{"gamma", "alpha"}) {
		t.Fatalf("soft-deleted report connectors=%v err=%v", names(revoked), err)
	}
	if _, err = db.ExecContext(ctx, "UPDATE reports SET deleted_at = NULL WHERE id = ?", "r-beta"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, "DELETE FROM report_acl WHERE report_id = ? AND subject_id = ?", "r-beta", "viewer"); err != nil {
		t.Fatal(err)
	}
	revoked, err = invokeAs("viewer", map[string]any{})
	if err != nil || !reflect.DeepEqual(names(revoked), []string{"gamma", "alpha"}) {
		t.Fatalf("revoked viewer connectors=%v err=%v", names(revoked), err)
	}
	// The SDK can page beyond the former static reader's 100-row cap.
	for index := 0; index < 137; index++ {
		name := fmt.Sprintf("owner-%03d", index)
		if _, err = db.ExecContext(ctx, `INSERT INTO connectors(name,driver,owner_id,status,etag,created_at,updated_at)
			VALUES(?, 'sqlite', 'owner-a', 'draft', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, name); err != nil {
			t.Fatal(err)
		}
	}
	large, err := invokeAs("owner-a", map[string]any{"limit": 150})
	if err != nil || len(large.Items) != 139 || large.PageLimit != 150 {
		t.Fatalf("large connector catalog rows=%d page=%+v err=%v", len(large.Items), large, err)
	}
	if strings.Contains(fmt.Sprint(large.Items), "private-token") {
		t.Fatal("large connector catalog exposed secret material")
	}
}
