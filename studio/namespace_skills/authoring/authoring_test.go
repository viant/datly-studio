package authoring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/runtime/accessprovider"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/viant/bindly/locator"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/readercomponent"
	"github.com/viant/datly-studio/sdk"
	directory "github.com/viant/datly-studio/studio/catalog"
	heads "github.com/viant/datly-studio/studio/namespace_skills/store_read"
	revisions "github.com/viant/datly-studio/studio/namespace_skills/store_revisions"
	stored "github.com/viant/datly-studio/studio/namespace_skills/store_write"
	namespaces "github.com/viant/datly-studio/studio/namespaces/store_read"
	runtimestatus "github.com/viant/datly-studio/studio/runtime/status"
	"github.com/viant/datly/bootstrap"
	gateway "github.com/viant/datly/gateway/http"
	"github.com/viant/datly/mcp"
	druntime "github.com/viant/datly/runtime"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	writerhandler "github.com/viant/datly/runtime/handler/writer"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/sql/dml"
	viewprovider "github.com/viant/datly/sql/reader/provider"
	dtag "github.com/viant/datly/tag"
	"github.com/viant/datly/typecatalog"
	resourceidentity "github.com/viant/forge/backend/resource"
	forgetypes "github.com/viant/forge/backend/types"
	"github.com/viant/mcp-protocol/authorization"
	namespaceext "github.com/viant/mcp-protocol/extension/namespaces"
	"github.com/viant/mcp-protocol/schema"
	"github.com/viant/x"
)

func TestNamespaceSkillAuthoringHTTP(t *testing.T) {
	ctx := context.Background()
	t.Setenv("STUDIO_NAMESPACE_ADMIN_ROLE", "")
	t.Setenv("STUDIO_MANAGEMENT_NAMESPACE", "")
	db := datatest.OpenSQLite(t, "namespace_skill_authoring", "studio")
	ns := namespaceaccess.ID("owner", "guides")
	foreign := namespaceaccess.ID("other", "guides")
	for _, pair := range [][2]string{{ns, "owner"}, {foreign, "other"}} {
		if _, err := db.Exec(`INSERT INTO namespaces(namespace_id,owner_id,name,title,status,visibility,mcp_enabled,etag,created_at,updated_at) VALUES(?,?,'guides','Guides','active','public',FALSE,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, pair[0], pair[1]); err != nil {
			t.Fatal(err)
		}
	}
	jwt := datatest.NewJWTFixture(t)
	resources := resource.New()
	for name, source := range map[string]any{heads.SkillDatlyResourceNamespace: heads.SkillDatlyResources, revisions.SkillDatlyResourceNamespace: revisions.SkillDatlyResources, stored.SkillDatlyResourceNamespace: stored.SkillDatlyResources, namespaces.NamespaceDatlyResourceNamespace: namespaces.NamespaceDatlyResources} {
		if err := resources.Register(name, source.(interface{ Open(string) (fs.File, error) })); err != nil {
			t.Fatal(err)
		}
	}
	types := datatest.StudioAuthorizationTypes(t)
	for _, value := range []any{stored.SkillStoreRules{}, stored.SkillRevisionRules{}} {
		if err := types.Register(typecatalog.TypeOriginPackage, x.NewType(reflect.TypeOf(value))); err != nil {
			t.Fatal(err)
		}
	}
	connector := &dsql.SQLComponent{DB: db}
	if err := connector.RegisterConnector("studio", db); err != nil {
		t.Fatal(err)
	}
	entries := []*registry.RegisteredComponent{datatest.AuthRegistration(t, db, jwt.Factory, resources)}
	for _, shape := range [][3]reflect.Type{
		{reflect.TypeFor[heads.SkillComponent](), reflect.TypeFor[heads.Input](), reflect.TypeFor[heads.Output]()},
		{reflect.TypeFor[revisions.SkillComponent](), reflect.TypeFor[revisions.Input](), reflect.TypeFor[revisions.Output]()},
		{reflect.TypeFor[namespaces.NamespaceComponent](), reflect.TypeFor[namespaces.Input](), reflect.TypeFor[namespaces.Output]()},
	} {
		entry, _, err := readercomponent.Compile(shape[0], "store_read", shape[1], shape[2], resources, connector, types)
		if err != nil {
			t.Fatal(err)
		}
		entry.Capabilities.Connector = connector
		entries = append(entries, entry)
	}
	register := func(holder, in, out reflect.Type, handler rhandler.TypedHandler, write bool) {
		t.Helper()
		field, _ := holder.FieldByName("Contract")
		tag, _, err := dtag.ParseComponent(field.Tag)
		if err != nil {
			t.Fatal(err)
		}
		component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackagePath: holder.PkgPath(), Tag: tag}).Resolve(in, out)
		if err != nil {
			t.Fatal(err)
		}
		artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: in, OutputType: out, Resources: resources, Types: types, CodecFactory: jwt.Factory})
		if err != nil {
			t.Fatal(err)
		}
		var providers []locator.Provider
		if len(artifact.ViewDependencies) > 0 {
			views, err := viewprovider.New(viewprovider.Config{Dependencies: artifact.ViewDependencies, Input: artifact.Input, SQL: connector})
			if err != nil {
				t.Fatal(err)
			}
			providers = append(providers, views)
		}
		if write {
			handler, err = writerhandler.New(component, in, out, "patch")
			if err != nil {
				t.Fatal(err)
			}
		}
		registration := registry.RegisteredComponent{Handler: handler, Providers: providers, DataSource: dml.Source{DB: db}}
		if !write && component.Settings != nil && component.Settings.IndependentChildTransactions {
			registration.DataSource = nil
		}
		registration.Capabilities.Connector = connector
		entry, err := artifact.Registration(registration)
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, entry)
	}
	register(reflect.TypeFor[stored.SkillComponent](), reflect.TypeFor[stored.Input](), reflect.TypeFor[stored.Output](), nil, true)
	register(reflect.TypeFor[ListComponent](), reflect.TypeFor[ListInput](), reflect.TypeFor[ListOutput](), custom.New(NewList()), false)
	register(reflect.TypeFor[GetComponent](), reflect.TypeFor[GetInput](), reflect.TypeFor[GetOutput](), custom.New(NewGet()), false)
	register(reflect.TypeFor[CreateComponent](), reflect.TypeFor[CreateInput](), reflect.TypeFor[GetOutput](), custom.New(NewCreate()), false)
	register(reflect.TypeFor[UpdateComponent](), reflect.TypeFor[UpdateInput](), reflect.TypeFor[GetOutput](), custom.New(NewUpdate()), false)
	register(reflect.TypeFor[DeleteComponent](), reflect.TypeFor[DeleteInput](), reflect.TypeFor[DeleteOutput](), custom.New(NewDelete()), false)
	register(reflect.TypeFor[RevisionsComponent](), reflect.TypeFor[RevisionsInput](), reflect.TypeFor[RevisionsOutput](), custom.New(NewRevisions()), false)
	register(reflect.TypeFor[PublishComponent](), reflect.TypeFor[PublishInput](), reflect.TypeFor[PublicationOutput](), custom.New(NewPublish()), false)
	register(reflect.TypeFor[UnpublishComponent](), reflect.TypeFor[UnpublishInput](), reflect.TypeFor[PublicationOutput](), custom.New(NewUnpublish()), false)
	register(reflect.TypeFor[runtimestatus.Component](), reflect.TypeFor[runtimestatus.Input](), reflect.TypeFor[runtimestatus.Output](), custom.New(runtimestatus.NewStatus()), false)
	register(reflect.TypeFor[directory.ListComponent](), reflect.TypeFor[directory.ListInput](), reflect.TypeFor[directory.ListOutput](), custom.New(directory.NewList()), false)
	register(reflect.TypeFor[directory.GetComponent](), reflect.TypeFor[directory.GetInput](), reflect.TypeFor[directory.GetOutput](), custom.New(directory.NewGet()), false)
	register(reflect.TypeFor[directory.ResourceComponent](), reflect.TypeFor[directory.ResourceInput](), reflect.TypeFor[directory.ResourceOutput](), custom.New(directory.NewResource()), false)
	runtime, err := druntime.NewRuntime(entries, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Shutdown(ctx)
	tools, err := mcp.New(mcp.Config{Components: entries, Invoker: runtime, Resources: resources})
	if err != nil {
		t.Fatal(err)
	}
	if names := tools.Catalog().ToolNames(); len(names) != 12 {
		t.Fatalf("private storage leaked into MCP: %v", names)
	}
	httpHandler := gateway.NewHandler(runtime, nil, "test")
	call := func(subject, namespace, op string, body any, status int, output any) {
		t.Helper()
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/namespace_skills."+op, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Studio-Namespace", namespace)
		if subject != "" {
			req.Header.Set("Authorization", jwt.Bearer(t, subject))
		}
		response := httptest.NewRecorder()
		httpHandler.ServeHTTP(response, req)
		if response.Code != status {
			t.Fatalf("%s: status=%d want=%d body=%s", op, response.Code, status, response.Body.String())
		}
		if output != nil {
			if err := json.Unmarshal(response.Body.Bytes(), output); err != nil {
				t.Fatalf("%s: %s %v", op, response.Body.String(), err)
			}
		}
	}
	catalogCall := func(subject, name string, arguments map[string]any, wantError bool) json.RawMessage {
		t.Helper()
		tool, ok := tools.Registry().ToolRegistry.Get(name)
		if !ok {
			t.Fatalf("catalog tool missing: %s", name)
		}
		caller := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, subject)})
		result, rpcErr := tool.Handler(caller, &schema.CallToolRequest{Method: schema.MethodToolsCall, Params: schema.CallToolRequestParams{Name: name, Arguments: arguments}})
		if rpcErr != nil || result == nil || (result.IsError != nil && *result.IsError) != wantError {
			t.Fatalf("catalog %s: %+v %v", name, result, rpcErr)
		}
		raw, err := json.Marshal(result.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	namespacePage := catalogCall("owner", "namespaces/list", map[string]any{}, false)
	var firstNamespacePage namespaceext.ListResult
	if err := json.Unmarshal(namespacePage, &firstNamespacePage); err != nil || len(firstNamespacePage.Namespaces) != 2 || firstNamespacePage.NextCursor != nil {
		t.Fatalf("namespace references: %s %v", namespacePage, err)
	}
	for _, ref := range firstNamespacePage.Namespaces {
		if ref.URI != "ns://"+ns && ref.URI != "ns://"+foreign {
			t.Fatalf("invalid namespace reference: %+v", ref)
		}
	}
	t.Run("configured management namespace resources", func(t *testing.T) {
		t.Setenv("STUDIO_MANAGEMENT_NAMESPACE", "dstudio")
		t.Setenv("STUDIO_MANAGEMENT_PROVIDER_ID", "studio-management")
		t.Setenv("STUDIO_MANAGEMENT_MCP_SERVICE", "studio-management")
		for _, name := range []string{"dstudio", "operations"} {
			t.Setenv("STUDIO_MANAGEMENT_NAMESPACE", name)
			raw := catalogCall("owner", "namespaces/list", map[string]any{}, false)
			var namespaces namespaceext.ListResult
			if err := json.Unmarshal(raw, &namespaces); err != nil || len(namespaces.Namespaces) != 3 {
				t.Fatalf("management namespace missing: %s %v", raw, err)
			}
			for _, kind := range []string{"", "window", "datasource", "mcp"} {
				raw = catalogCall("owner", "namespaces/get", map[string]any{"uri": "ns://" + name, "kind": kind}, false)
				var page namespaceext.GetResult
				want := 1
				if kind == "" {
					want = 2
				}
				if kind == "mcp" {
					want = 0
				}
				if err := json.Unmarshal(raw, &page); err != nil || len(page.Resources) != want || page.NextCursor != nil {
					t.Fatalf("management resource kinds: %s %v", raw, err)
				}
			}
			window := catalogCall("owner", "resource/get", map[string]any{"uri": "window://" + name + "/namespaces"}, false)
			datasource := catalogCall("owner", "resource/get", map[string]any{"uri": "datasource://" + name + "/namespaces"}, false)
			var envelope forgetypes.WindowResourceEnvelope
			if err := json.Unmarshal(window, &envelope); err != nil {
				t.Fatal(err)
			}
			if err := envelope.Validate(); err != nil {
				t.Fatalf("wire window: %v", err)
			}
			for _, variant := range envelope.Variants {
				if variant.DataSourceResources["namespaces"].ContentFingerprint != resourceidentity.ContentFingerprint(datasource) {
					t.Fatal("MCP serialization changed referenced datasource bytes")
				}
			}
			catalogCall("owner", "resource/get", map[string]any{"uri": "window://foreign/namespaces"}, true)
		}
	})
	t.Run("namespace catalog pagination and visibility", func(t *testing.T) {
		// Cross both the private reader's 500-row boundary and the public
		// 32-entry page boundary. Private rows must be filtered before paging.
		ids := make([]string, 513)
		for i := range ids {
			name := fmt.Sprintf("pagination-%03d", i)
			ids[i] = namespaceaccess.ID("owner", name)
			if _, err := db.Exec(`INSERT INTO namespaces(namespace_id,owner_id,name,title,description,status,visibility,mcp_enabled,etag,created_at,updated_at) VALUES(?,'owner',?,?,'Pagination fixture','active','private',FALSE,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, ids[i], name, name); err != nil {
				t.Fatal(err)
			}
		}
		t.Cleanup(func() {
			for _, id := range ids {
				if _, err := db.Exec(`DELETE FROM namespaces WHERE namespace_id=?`, id); err != nil {
					t.Error(err)
				}
			}
		})
		seen := map[string]bool{}
		lastURI := ""
		var firstCursor string
		arguments := map[string]any{}
		for pageNumber := 0; ; pageNumber++ {
			if pageNumber > 20 {
				t.Fatal("namespace pagination did not terminate")
			}
			raw := catalogCall("owner", "namespaces/list", arguments, false)
			var page namespaceext.ListResult
			if err := json.Unmarshal(raw, &page); err != nil || len(page.Namespaces) == 0 || len(page.Namespaces) > 32 {
				t.Fatalf("invalid namespace page: %s %v", raw, err)
			}
			for _, ref := range page.Namespaces {
				if seen[ref.URI] || ref.URI <= lastURI {
					t.Fatalf("duplicate or unsorted namespace: %s", ref.URI)
				}
				if ref.URI != "ns://"+ns && ref.URI != "ns://"+foreign && ref.Description != "Pagination fixture" {
					t.Fatalf("namespace description missing: %+v", ref)
				}
				seen[ref.URI], lastURI = true, ref.URI
			}
			if page.NextCursor == nil {
				break
			}
			if len(page.Namespaces) != 32 {
				t.Fatal("nonfinal namespace page was truncated")
			}
			if firstCursor == "" {
				firstCursor = *page.NextCursor
			}
			arguments = map[string]any{"cursor": *page.NextCursor}
		}
		if len(seen) != len(ids)+2 {
			t.Fatalf("namespace pagination lost entries: got %d want %d", len(seen), len(ids)+2)
		}
		var outsider namespaceext.ListResult
		raw := catalogCall("outsider", "namespaces/list", map[string]any{}, false)
		if err := json.Unmarshal(raw, &outsider); err != nil || len(outsider.Namespaces) != 2 || outsider.NextCursor != nil {
			t.Fatalf("private namespaces affected visible pagination: %s %v", raw, err)
		}
		catalogCall("outsider", "namespaces/list", map[string]any{"cursor": firstCursor}, true)
		catalogCall("owner", "namespaces/list", map[string]any{"cursor": "invalid"}, true)
		if _, err := db.Exec(`UPDATE namespaces SET description='Changed snapshot' WHERE namespace_id=?`, ids[0]); err != nil {
			t.Fatal(err)
		}
		catalogCall("owner", "namespaces/list", map[string]any{"cursor": firstCursor}, true)
	})
	if _, err := db.Exec(`UPDATE namespaces SET visibility='private' WHERE namespace_id=?`, ns); err != nil {
		t.Fatal(err)
	}
	visiblePage := catalogCall("outsider", "namespaces/list", map[string]any{}, false)
	var visibleNamespacePage namespaceext.ListResult
	if err := json.Unmarshal(visiblePage, &visibleNamespacePage); err != nil || len(visibleNamespacePage.Namespaces) != 1 || visibleNamespacePage.Namespaces[0].URI != "ns://"+foreign {
		t.Fatalf("namespace visibility: %s %v", visiblePage, err)
	}
	catalogCall("outsider", "namespaces/get", map[string]any{"uri": "ns://" + ns}, true)
	catalogCall("outsider", "resource/get", map[string]any{"uri": directory.VirtualURI(ns, "guides")}, true)
	// An authenticated principal with no visible namespaces receives an empty
	// collection, not an access error, null, or a cursor to a nonexistent page.
	if _, err := db.Exec(`UPDATE namespaces SET visibility='private' WHERE namespace_id=?`, foreign); err != nil {
		t.Fatal(err)
	}
	emptyPage := catalogCall("outsider", "namespaces/list", map[string]any{}, false)
	var emptyWire map[string]json.RawMessage
	if err := json.Unmarshal(emptyPage, &emptyWire); err != nil || string(emptyWire["namespaces"]) != "[]" || emptyWire["nextCursor"] != nil {
		t.Fatalf("empty namespace list must be a successful array without a cursor: %s %v", emptyPage, err)
	}
	if _, err := db.Exec(`UPDATE namespaces SET visibility='public' WHERE namespace_id=?`, foreign); err != nil {
		t.Fatal(err)
	}
	raw := catalogCall("owner", "namespaces/get", map[string]any{"uri": "ns://" + ns}, false)
	var initial namespaceext.GetResult
	if err := json.Unmarshal(raw, &initial); err != nil || len(initial.Resources) != 0 {
		t.Fatalf("disabled namespace resources: %s %v", raw, err)
	}
	bundle := sdk.NamespaceSkillBundle{Content: "---\nname: guide\ndescription: Namespace guide\n---\nFirst version.", Files: []sdk.NamespaceSkillFile{{Path: "references/example.md", Content: "Example"}}}
	var created GetOutput
	call("owner", ns, "create", map[string]any{"bundle": bundle}, 200, &created)
	if created.Skill == nil || created.Revision == nil || created.Skill.ETag != 1 || created.Skill.PublishedRevision != 0 || created.Revision.Bundle.Content != bundle.Content {
		t.Fatalf("create: %+v", created)
	}
	tool, ok := tools.Registry().ToolRegistry.Get("studio.sdk.namespace_skills.get")
	if !ok {
		t.Fatal("namespace skill MCP get is missing")
	}
	callContext := context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, "owner")})
	result, rpcErr := tool.Handler(callContext, &schema.CallToolRequest{Method: schema.MethodToolsCall, Params: schema.CallToolRequestParams{Name: "studio.sdk.namespace_skills.get", Arguments: map[string]any{"namespaceId": ns, "skillId": created.Skill.SkillID}}})
	if rpcErr != nil || result == nil || result.IsError != nil && *result.IsError {
		t.Fatalf("MCP get: %+v %v", result, rpcErr)
	}
	structured, err := json.Marshal(result.StructuredContent)
	if err != nil || !bytes.Contains(structured, []byte("First version.")) {
		t.Fatalf("MCP revision content: %s %v", structured, err)
	}
	id := created.Skill.SkillID
	call("owner", ns, "create", map[string]any{"bundle": bundle}, 409, nil)
	call("other", ns, "list", map[string]any{}, 403, nil)
	call("owner", foreign, "get", map[string]any{"skillId": id}, 403, nil)
	call("", ns, "get", map[string]any{"skillId": id}, 401, nil)
	var listed ListOutput
	call("owner", ns, "list", map[string]any{"query": "guide", "limit": 1}, 200, &listed)
	if len(listed.Items) != 1 || listed.Items[0].SkillID != id {
		t.Fatalf("list: %+v", listed)
	}
	call("owner", ns, "list", map[string]any{"offset": 1}, 200, &listed)
	if len(listed.Items) != 0 {
		t.Fatal("pagination ignored")
	}
	bundle.Content = "---\nname: guide\ndescription: Updated guide\n---\nSecond version."
	var updated GetOutput
	call("owner", ns, "update", map[string]any{"skillId": id, "etag": 1, "bundle": bundle}, 200, &updated)
	if updated.Skill.ETag != 2 || updated.Skill.LatestRevision != 2 {
		t.Fatalf("update: %+v", updated.Skill)
	}
	call("owner", ns, "update", map[string]any{"skillId": id, "etag": 1, "bundle": bundle}, 409, nil)
	var old GetOutput
	call("owner", ns, "get", map[string]any{"skillId": id, "revision": 1}, 200, &old)
	if old.Revision.Bundle.Content != created.Revision.Bundle.Content {
		t.Fatal("revision 1 changed")
	}
	var history RevisionsOutput
	call("owner", ns, "revisions", map[string]any{"skillId": id}, 200, &history)
	if len(history.Items) != 2 || history.Items[0].Revision != 2 {
		t.Fatalf("history: %+v", history)
	}

	var failReload atomic.Bool
	var observedRevision atomic.Int64
	var connectionStatus atomic.Int64
	connectionStatus.Store(http.StatusOK)
	var connectionURL atomic.Value
	connectionURL.Store("https://namespace.example/mcp")
	admin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Studio-Runtime-Token") != "fixture-token" {
			http.Error(w, "unauthorized", 401)
			return
		}
		if r.URL.Path == "/_studio/connection" {
			if status := int(connectionStatus.Load()); status != http.StatusOK {
				http.Error(w, "fixture runtime unavailable", status)
				return
			}
			if r.URL.Query().Get("namespaceId") != ns {
				http.Error(w, "wrong namespace", 400)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"namespaceId": ns, "config": map[string]any{"name": "guides", "transport": map[string]any{"type": "streamable", "url": connectionURL.Load()}}})
			return
		}
		if r.URL.Path == "/_studio/status" {
			if r.URL.Query().Get("namespaceId") != ns {
				http.Error(w, "wrong namespace", 400)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"namespaceId": ns, "mcpUrl": "http://127.0.0.1:12345/mcp", "status": "ready", "revision": 1})
			return
		}
		var payload struct {
			NamespaceID string `json:"namespaceId"`
			Generation  int64  `json:"generation"`
		}
		if json.NewDecoder(r.Body).Decode(&payload) != nil || payload.NamespaceID != ns || payload.Generation != 0 {
			http.Error(w, "wrong reload", 400)
			return
		}
		var selected int
		if err := db.QueryRowContext(r.Context(), `SELECT published_revision FROM namespace_skills WHERE namespace_id=? AND skill_id=?`, ns, id).Scan(&selected); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		observedRevision.Store(int64(selected))
		if failReload.Swap(false) {
			http.Error(w, "fixture activation failure", 503)
			return
		}
		w.WriteHeader(200)
	}))
	defer admin.Close()
	t.Setenv("STUDIO_DYNAMIC_HTTP_URL", admin.URL)
	t.Setenv("STUDIO_RUNTIME_ADMIN_TOKEN", "fixture-token")
	statusRequest := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/runtime.status", bytes.NewBufferString("{}"))
	statusRequest.Header.Set("Content-Type", "application/json")
	statusRequest.Header.Set("X-Studio-Namespace", ns)
	statusRequest.Header.Set("Authorization", jwt.Bearer(t, "owner"))
	statusRequest = statusRequest.WithContext(accessprovider.WithProvider(ctx, namespaceStatusFacts{}))
	statusResponse := httptest.NewRecorder()
	httpHandler.ServeHTTP(statusResponse, statusRequest)
	if statusResponse.Code != 200 {
		t.Fatalf("native empty namespace status: %d %s", statusResponse.Code, statusResponse.Body.String())
	}
	statusTool, ok := tools.Registry().ToolRegistry.Get("studio.sdk.runtime.status")
	if !ok {
		t.Fatal("runtime status tool missing")
	}
	statusCtx := accessprovider.WithProvider(context.WithValue(ctx, authorization.TokenKey, &authorization.Token{Token: jwt.Bearer(t, "owner")}), namespaceStatusFacts{})
	statusResult, statusErr := statusTool.Handler(statusCtx, &schema.CallToolRequest{Method: schema.MethodToolsCall, Params: schema.CallToolRequestParams{Name: "studio.sdk.runtime.status", Arguments: map[string]any{"namespaceId": ns}}})
	if statusErr != nil || statusResult == nil || statusResult.IsError != nil && *statusResult.IsError {
		t.Fatalf("native MCP empty namespace status: %+v %v", statusResult, statusErr)
	}
	if _, err := db.Exec(`UPDATE namespaces SET mcp_enabled=TRUE WHERE namespace_id=?`, ns); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"", "mcp"} {
		raw := catalogCall("owner", "namespaces/get", map[string]any{"uri": "ns://" + ns, "kind": kind}, false)
		var page namespaceext.GetResult
		if err := json.Unmarshal(raw, &page); err != nil || len(page.Resources) != 1 || page.NextCursor != nil || page.Resources[0].URI != directory.VirtualURI(ns, "guides") {
			t.Fatalf("virtual resource page: %s %v", raw, err)
		}
	}
	configJSON := catalogCall("owner", "resource/get", map[string]any{"uri": directory.VirtualURI(ns, "guides")}, false)
	var configuration struct {
		Name      string `json:"name"`
		Transport struct {
			Type string `json:"type"`
			URL  string `json:"url"`
		} `json:"transport"`
	}
	if err := json.Unmarshal(configJSON, &configuration); err != nil || configuration.Name != "guides" || configuration.Transport.URL != "https://namespace.example/mcp" {
		t.Fatalf("virtual JSON configuration: %s %v", configJSON, err)
	}
	t.Run("connection catalog follows active runtime", func(t *testing.T) {
		connectionStatus.Store(http.StatusNotFound)
		raw := catalogCall("owner", "namespaces/get", map[string]any{"uri": "ns://" + ns}, false)
		var page namespaceext.GetResult
		if err := json.Unmarshal(raw, &page); err != nil || page.Resources == nil || len(page.Resources) != 0 || page.NextCursor != nil {
			t.Fatalf("stopped runtime advertised a connection: %s %v", raw, err)
		}
		catalogCall("owner", "resource/get", map[string]any{"uri": directory.VirtualURI(ns, "guides")}, true)
		// An unavailable admin service is an error, not evidence of an empty catalog.
		connectionStatus.Store(http.StatusServiceUnavailable)
		catalogCall("owner", "namespaces/get", map[string]any{"uri": "ns://" + ns}, true)
		connectionURL.Store("https://namespace.example:9443/mcp")
		connectionStatus.Store(http.StatusOK)
		raw = catalogCall("owner", "namespaces/get", map[string]any{"uri": "ns://" + ns}, false)
		if err := json.Unmarshal(raw, &page); err != nil || len(page.Resources) != 1 {
			t.Fatalf("restarted runtime missing: %s %v", raw, err)
		}
		raw = catalogCall("owner", "resource/get", map[string]any{"uri": directory.VirtualURI(ns, "guides")}, false)
		if err := json.Unmarshal(raw, &configuration); err != nil || configuration.Transport.URL != "https://namespace.example:9443/mcp" {
			t.Fatalf("connection configuration did not update: %s %v", raw, err)
		}
	})
	var publication PublicationOutput
	call("owner", ns, "publish", map[string]any{"skillId": id, "etag": 2, "revision": 1}, 200, &publication)
	if publication.Skill.PublishedRevision != 1 || publication.Skill.ETag != 3 || publication.PublicationStatus != "active" || observedRevision.Load() != 1 {
		t.Fatalf("publication did not commit before reload: %+v", publication)
	}
	call("owner", ns, "delete", map[string]any{"skillId": id, "etag": 3}, 409, nil)
	failReload.Store(true)
	call("owner", ns, "publish", map[string]any{"skillId": id, "etag": 3, "revision": 2}, 503, nil)
	var restored GetOutput
	call("owner", ns, "get", map[string]any{"skillId": id}, 200, &restored)
	if restored.Skill.PublishedRevision != 1 || restored.Skill.ETag != 5 || observedRevision.Load() != 1 {
		t.Fatalf("failed activation did not restore previous publication: %+v", restored.Skill)
	}
	call("owner", ns, "unpublish", map[string]any{"skillId": id, "etag": 5}, 200, &publication)
	if publication.Skill.PublishedRevision != 0 || publication.Skill.ETag != 6 || publication.PublicationStatus != "unpublished" || observedRevision.Load() != 0 {
		t.Fatalf("unpublish: %+v", publication)
	}
	call("owner", ns, "delete", map[string]any{"skillId": id, "etag": 6}, 200, nil)

	call("owner", ns, "get", map[string]any{"skillId": id}, 404, nil)
	var count int
	if err = db.QueryRow(`SELECT COUNT(*) FROM components`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("unexpected hidden component: %d %v", count, err)
	}
}

type namespaceStatusFacts struct{}

func (namespaceStatusFacts) Resolve(ctx context.Context) (authz.Facts, error) {
	if oauth.Bearer(ctx) == "" {
		return authz.Facts{}, authz.ErrDenied
	}
	return authz.Facts{Subject: "owner", Issuer: "test", Tenant: "test", Roles: []string{"XYZ"}, ValidUntil: time.Now().Add(time.Minute)}, nil
}
