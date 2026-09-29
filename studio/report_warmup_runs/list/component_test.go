package list

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/viant/bindly/locator"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/sdk"
	expired "github.com/viant/datly-studio/studio/report_warmup_runs/store_expired"
	storedreader "github.com/viant/datly-studio/studio/report_warmup_runs/store_read"
	storedwriter "github.com/viant/datly-studio/studio/report_warmup_runs/store_write"
	guard "github.com/viant/datly-studio/studio/reports/publish_guard"
	"github.com/viant/datly/bootstrap"
	gateway "github.com/viant/datly/gateway/http"
	"github.com/viant/datly/gateway/openapi"
	"github.com/viant/datly/gateway/openapi/openapi3"
	"github.com/viant/datly/mcp"
	druntime "github.com/viant/datly/runtime"
	rhandler "github.com/viant/datly/runtime/handler"
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

func TestWarmupListSDKDatlyHTTPMCPAndRecovery(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "sdk_warmup_list", "studio")
	jwt := datatest.NewJWTFixture(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	old := now.Add(-warmupRunTimeout - time.Minute)
	if err := datatest.Hydrate(ctx, db,
		datatest.Table{Name: "connectors", Rows: []datatest.Row{{"name": "main", "driver": "sqlite", "owner_id": "alice", "status": "active", "created_at": now, "updated_at": now}}},
		datatest.Table{Name: "namespaces", Rows: []datatest.Row{{"owner_id": "alice", "name": "general", "title": "General", "status": "active", "created_at": now, "updated_at": now}}},
		datatest.Table{Name: "components", Rows: []datatest.Row{
			{"id": "r1", "slug": "first", "title": "First", "owner_id": "alice", "status": "active", "default_connector_name": "main", "namespace": "general", "component_scope": "reports/first", "component_name": "first", "created_at": now, "updated_at": now},
			{"id": "r2", "slug": "second", "title": "Second", "owner_id": "alice", "status": "active", "default_connector_name": "main", "namespace": "general", "component_scope": "reports/second", "component_name": "second", "created_at": now, "updated_at": now},
		}},
		datatest.Table{Name: "report_acl", Rows: []datatest.Row{{"report_id": "r1", "subject_type": "user", "subject_id": "publisher", "can_view": true, "can_publish": true}, {"report_id": "r1", "subject_type": "user", "subject_id": "viewer", "can_view": true}}},
		datatest.Table{Name: "report_versions", Rows: []datatest.Row{
			{"report_id": "r1", "version_no": 1, "state": "draft", "authoring_mode": "dql", "component_spec_json": "{}", "type_manifest_json": "{}", "spec_format_version": "1", "spec_hash": "hash", "compile_status": "valid", "datly_version": "v1", "compiler_version": "v1", "source_revision": 1, "created_by": "alice", "created_at": now},
			{"report_id": "r2", "version_no": 1, "state": "draft", "authoring_mode": "dql", "component_spec_json": "{}", "type_manifest_json": "{}", "spec_format_version": "1", "spec_hash": "hash", "compile_status": "valid", "datly_version": "v1", "compiler_version": "v1", "source_revision": 1, "created_by": "alice", "created_at": now},
		}},
	); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		id, report, status string
		requestedAt        time.Time
	}{
		{"old-a", "r1", "accepted", old.Add(-time.Second)},
		{"old-b", "r1", "running", old},
		{"recent", "r1", "accepted", now},
		{"foreign-old", "r2", "accepted", old},
	} {
		if _, err := db.ExecContext(ctx, `INSERT INTO report_warmup_runs
 (run_id,report_id,version_no,source_revision,spec_hash,plan_key,active_key,status,requested_by,target_json,requested_at,updated_at,updated_by)
 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, item.id, item.report, 1, 1, "hash", "plan", item.id, item.status, "alice", `{}`, item.requestedAt, item.requestedAt, "alice"); err != nil {
			t.Fatal(err)
		}
	}
	for index := 0; index < 105; index++ {
		id := fmt.Sprintf("foreign-batch-%03d", index)
		if _, err := db.ExecContext(ctx, `INSERT INTO report_warmup_runs
 (run_id,report_id,version_no,source_revision,spec_hash,plan_key,active_key,status,requested_by,target_json,requested_at,updated_at,updated_by)
 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, id, "r2", 1, 1, "hash", "plan", id, "accepted", "alice", `{}`, old, old, "alice"); err != nil {
			t.Fatal(err)
		}
	}
	resources := resource.New()
	for _, item := range []struct {
		name string
		fs   fs.FS
	}{
		{guard.ReportDatlyResourceNamespace, guard.ReportDatlyResources},
		{expired.WarmupRunDatlyResourceNamespace, expired.WarmupRunDatlyResources},
		{storedreader.WarmupRunDatlyResourceNamespace, storedreader.WarmupRunDatlyResources},
		{storedwriter.WarmupRunDatlyResourceNamespace, storedwriter.WarmupRunDatlyResources},
	} {
		if err := resources.Register(item.name, item.fs); err != nil {
			t.Fatal(err)
		}
	}
	authEntry := datatest.AuthRegistration(t, db, jwt.Factory, resources)
	connector := &dsql.SQLComponent{DB: db}
	if err := connector.RegisterConnector("studio", db); err != nil {
		t.Fatal(err)
	}
	compile := func(holder reflect.Type, fieldName string, inputType, outputType reflect.Type, mode string, handler rhandler.TypedHandler) *registry.RegisteredComponent {
		t.Helper()
		field, ok := holder.FieldByName(fieldName)
		if !ok {
			t.Fatalf("%s.%s missing", holder, fieldName)
		}
		metadata, present, parseErr := dtag.ParseComponent(field.Tag)
		if parseErr != nil || !present {
			t.Fatalf("%s metadata: %v", holder, parseErr)
		}
		component, resolveErr := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackageName: "list", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(inputType, outputType)
		if resolveErr != nil {
			t.Fatal(resolveErr)
		}
		artifact, buildErr := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: inputType, OutputType: outputType,
			Resources: resources, Types: datatest.StudioAuthorizationTypes(t), CodecFactory: jwt.Factory,
			Handler: handler, HandlerOwnedOutput: handler != nil})
		if buildErr != nil {
			t.Fatal(buildErr)
		}
		registration := registry.RegisteredComponent{}
		switch mode {
		case "reader":
			execution, compileErr := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: connector})
			if compileErr != nil {
				t.Fatal(compileErr)
			}
			registration.Reader = execution
		case "writer":
			var providers []locator.Provider
			if len(artifact.ViewDependencies) > 0 {
				views, providerErr := viewprovider.New(viewprovider.Config{Dependencies: artifact.ViewDependencies, Input: artifact.Input, SQL: connector})
				if providerErr != nil {
					t.Fatal(providerErr)
				}
				providers = append(providers, views)
			}
			writer, writerErr := writerhandler.New(component, inputType, outputType, "patch")
			if writerErr != nil {
				t.Fatal(writerErr)
			}
			registration.Handler, registration.Providers, registration.DataSource = writer, providers, dml.Source{DB: db}
			registration.Capabilities.Connector = connector
		}
		entry, registerErr := artifact.Registration(registration)
		if registerErr != nil {
			t.Fatal(registerErr)
		}
		return entry
	}
	guardEntry := compile(reflect.TypeFor[guard.ReportComponent](), "Contract", reflect.TypeFor[guard.Input](), reflect.TypeFor[guard.Output](), "reader", nil)
	expiredEntry := compile(reflect.TypeFor[expired.WarmupRunComponent](), "Contract", reflect.TypeFor[expired.Input](), reflect.TypeFor[expired.Output](), "reader", nil)
	readEntry := compile(reflect.TypeFor[storedreader.WarmupRunComponent](), "Contract", reflect.TypeFor[storedreader.Input](), reflect.TypeFor[storedreader.Output](), "reader", nil)
	writeEntry := compile(reflect.TypeFor[storedwriter.WarmupRunComponent](), "Contract", reflect.TypeFor[storedwriter.Input](), reflect.TypeFor[storedwriter.Output](), "writer", nil)
	listHandler, err := (Component{}).DatlyHandler("NewWarmupList")()
	if err != nil {
		t.Fatal(err)
	}
	listEntry := compile(reflect.TypeFor[Component](), "Contract", reflect.TypeFor[Input](), reflect.TypeFor[Output](), "custom", listHandler)
	entries := []*registry.RegisteredComponent{authEntry, guardEntry, expiredEntry, readEntry, writeEntry, listEntry}
	for _, entry := range []*registry.RegisteredComponent{guardEntry, expiredEntry, readEntry, writeEntry} {
		if !entry.Component.Routes[0].Internal || len(entry.Component.Routes[0].MCP) != 0 {
			t.Fatalf("private warmup component is exposed: %+v", entry.Component.Routes[0])
		}
	}
	for _, entry := range entries {
		entry.Capabilities.Connector = connector
	}
	runtime, err := druntime.NewRuntime(entries, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Shutdown(ctx) })
	tools, err := mcp.New(mcp.Config{Components: entries, Invoker: runtime, Resources: resources})
	if err != nil {
		t.Fatal(err)
	}
	if names := tools.Catalog().ToolNames(); !reflect.DeepEqual(names, []string{"studio.sdk.versions.warmup_list"}) {
		t.Fatalf("MCP tools=%v", names)
	}
	request := func(subject, reportID string, input string) *http.Request {
		body := `{"reportId":"` + reportID + `","versionNo":1,"input":` + input + `}`
		req := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/versions.warmup_list", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		if subject != "" {
			req.Header.Set("Authorization", jwt.Bearer(t, subject))
		}
		return req
	}
	httpHandler := gateway.NewHandler(runtime, nil, "test")
	// Owning both workspaces must not allow a selected workspace to read the other.
	selectedID := namespaceaccess.ID("alice", "other")
	if _, err := db.ExecContext(ctx, `INSERT INTO namespaces(owner_id,name,title,status,namespace_id,created_at,updated_at) VALUES('alice','other','Other','active',?,?,?)`, selectedID, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE namespaces SET namespace_id=? WHERE owner_id='alice' AND name='general'`, namespaceaccess.ID("alice", "general")); err != nil {
		t.Fatal(err)
	}
	for _, selection := range []string{selectedID, "invalid"} {
		deniedRequest := request("alice", "r1", `{}`)
		deniedRequest.Header.Set("X-Studio-Namespace", selection)
		denied := httptest.NewRecorder()
		httpHandler.ServeHTTP(denied, deniedRequest)
		if denied.Code == http.StatusOK {
			t.Fatalf("out-of-scope warmup list returned %d: %s", denied.Code, denied.Body.String())
		}
	}
	var unchanged string
	if err := db.QueryRowContext(ctx, "SELECT status FROM report_warmup_runs WHERE run_id='old-a'").Scan(&unchanged); err != nil || unchanged != "accepted" {
		t.Fatalf("denial performed expiry recovery: %q %v", unchanged, err)
	}
	viewer := httptest.NewRecorder()
	httpHandler.ServeHTTP(viewer, request("viewer", "r1", `{}`))
	if viewer.Code != http.StatusNotFound {
		t.Fatalf("view-only warmup list status=%d body=%s", viewer.Code, viewer.Body.String())
	}
	var before string
	if err = db.QueryRowContext(ctx, "SELECT status FROM report_warmup_runs WHERE run_id='foreign-old'").Scan(&before); err != nil || before != "accepted" {
		t.Fatalf("unauthorized list triggered global recovery: status=%q err=%v", before, err)
	}
	allowed := httptest.NewRecorder()
	httpHandler.ServeHTTP(allowed, request("publisher", "r1", `{}`))
	if allowed.Code != http.StatusOK {
		t.Fatalf("publisher list status=%d body=%s", allowed.Code, allowed.Body.String())
	}
	var page sdk.WarmupRunPage
	if err = json.Unmarshal(allowed.Body.Bytes(), &page); err != nil || len(page.Items) != 3 || page.Limit != 20 || page.Offset != 0 || page.Items[0].RunID != "recent" {
		t.Fatalf("warmup page=%+v err=%v body=%s", page, err, allowed.Body.String())
	}
	for _, id := range []string{"old-a", "old-b", "foreign-old"} {
		var status, updatedBy, diagnostics string
		if err = db.QueryRowContext(ctx, "SELECT status,updated_by,diagnostics_json FROM report_warmup_runs WHERE run_id=?", id).Scan(&status, &updatedBy, &diagnostics); err != nil || status != "failed" || updatedBy != sdk.SystemPrincipal().Subject || !bytes.Contains([]byte(diagnostics), []byte("warmup_expired")) {
			t.Fatalf("expired %s status=%q actor=%q diagnostics=%q err=%v", id, status, updatedBy, diagnostics, err)
		}
	}
	var recoveredForeign int
	if err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM report_warmup_runs WHERE report_id='r2' AND status='failed' AND diagnostics_json LIKE '%warmup_expired%'").Scan(&recoveredForeign); err != nil || recoveredForeign != 106 {
		t.Fatalf("global recovery across 100-row pages=%d err=%v", recoveredForeign, err)
	}
	var recent string
	if err = db.QueryRowContext(ctx, "SELECT status FROM report_warmup_runs WHERE run_id='recent'").Scan(&recent); err != nil || recent != "accepted" {
		t.Fatalf("recent run status=%q err=%v", recent, err)
	}
	bounded := httptest.NewRecorder()
	httpHandler.ServeHTTP(bounded, request("publisher", "r1", `{"limit":1,"offset":1}`))
	if bounded.Code != http.StatusOK {
		t.Fatalf("bounded list status=%d body=%s", bounded.Code, bounded.Body.String())
	}
	if err = json.Unmarshal(bounded.Body.Bytes(), &page); err != nil || len(page.Items) != 1 || page.Limit != 1 || page.Offset != 1 {
		t.Fatalf("bounded page=%+v err=%v", page, err)
	}
	negative := httptest.NewRecorder()
	httpHandler.ServeHTTP(negative, request("publisher", "r1", `{"offset":-1}`))
	if negative.Code != http.StatusBadRequest {
		t.Fatalf("negative offset status=%d body=%s", negative.Code, negative.Body.String())
	}
	missing := httptest.NewRecorder()
	httpHandler.ServeHTTP(missing, request("publisher", "absent", `{}`))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing report status=%d body=%s", missing.Code, missing.Body.String())
	}
	unauthenticated := httptest.NewRecorder()
	httpHandler.ServeHTTP(unauthenticated, request("", "r1", `{}`))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("missing JWT status=%d body=%s", unauthenticated.Code, unauthenticated.Body.String())
	}
	document, err := (openapi.Generator{}).Generate(ctx, openapi.Request{Info: openapi3.Info{Title: "Studio SDK", Version: "1"}, Components: entries, Routes: []spec.RouteRef{{Method: "POST", Path: "/v1/studio/sdk/versions.warmup_list"}}})
	if err != nil || document.Paths["/v1/studio/sdk/versions.warmup_list"].Post == nil {
		t.Fatalf("OpenAPI warmup list route: %v", err)
	}
	for _, path := range []string{"/_studio/reports/publish-guard", "/_studio/report-warmup-run-store/expired", "/_studio/report-warmup-run-store/read", "/_studio/report-warmup-run-store/write"} {
		if document.Paths[path] != nil {
			t.Fatalf("private component %s appears in OpenAPI", path)
		}
		private := httptest.NewRecorder()
		httpHandler.ServeHTTP(private, httptest.NewRequest(http.MethodGet, path, nil))
		if private.Code == http.StatusOK {
			t.Fatalf("private component %s publicly reachable", path)
		}
	}
	tool, _ := tools.Registry().ToolRegistry.Get("studio.sdk.versions.warmup_list")
	ownerContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, "alice")})
	scopedResult, scopedErr := tool.Handler(ownerContext, &schema.CallToolRequest{Method: schema.MethodToolsCall, Params: schema.CallToolRequestParams{Name: "studio.sdk.versions.warmup_list", Arguments: map[string]any{"reportId": "r1", "versionNo": 1, "namespaceId": selectedID}}})
	if scopedErr == nil && scopedResult != nil && (scopedResult.IsError == nil || !*scopedResult.IsError) {
		t.Fatalf("MCP disclosed another workspace's warmup runs: %+v", scopedResult)
	}

	callContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, "publisher")})
	result, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall, Params: schema.CallToolRequestParams{Name: "studio.sdk.versions.warmup_list", Arguments: map[string]any{"reportId": "r1", "versionNo": 1, "input": map[string]any{"limit": 1}}}})
	if rpcErr != nil || result == nil || result.IsError != nil && *result.IsError {
		t.Fatalf("MCP warmup list result=%+v err=%v", result, rpcErr)
	}
	structured, err := json.Marshal(result.StructuredContent)
	if err != nil || !bytes.Contains(structured, []byte(`"limit":1`)) {
		t.Fatalf("MCP warmup list=%s err=%v", structured, err)
	}
	if _, err = db.ExecContext(ctx, "DELETE FROM report_acl WHERE report_id = ? AND subject_id = ?", "r1", "publisher"); err != nil {
		t.Fatal(err)
	}
	revoked := httptest.NewRecorder()
	httpHandler.ServeHTTP(revoked, request("publisher", "r1", `{}`))
	if revoked.Code != http.StatusNotFound {
		t.Fatalf("revoked publisher status=%d body=%s", revoked.Code, revoked.Body.String())
	}
}
