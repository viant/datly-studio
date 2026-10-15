package host

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	mcpconfig "github.com/viant/mcp/client/config"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/namespaceskill"
	"github.com/viant/datly-studio/internal/runtimeadmin"
	"github.com/viant/datly-studio/runtime/namespacemcp"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	mcpschema "github.com/viant/mcp-protocol/schema"
	_ "modernc.org/sqlite"
)

func TestNamespaceFactoryBuildsScopedRuntimeCatalogsAndReloads(t *testing.T) {
	for _, mode := range []string{"separate", "shared"} {
		t.Run(mode, func(t *testing.T) { testNamespaceFactory(t, mode) })
	}
}
func testNamespaceFactory(t *testing.T, mode string) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "studio.db")
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO connectors(name,driver,dsn_template,owner_id,status,etag,created_at,updated_at) VALUES('shared','sqlite',?,'owner','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
 INSERT INTO runtime_generations(generation_no,source_revision,status,report_count,build_manifest_json,requested_by,requested_at) VALUES(1,'fixture','active',2,'{}','owner',CURRENT_TIMESTAMP);`, dsn); err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	for _, name := range []string{"alpha", "beta"} {
		id := namespaceaccess.ID("owner", name)
		ids[name] = id
		dql := `#package('github.com/viant/datly-studio/dynamic/` + name + `')
#setting($_ = $connector('shared'))
#setting($_ = $route('/` + name + `','GET'))
#setting($_ = $mcp('` + name + `.read','Read fixture'))
#define($_ = $Rows<[]*Row>(output/view))
SELECT rows.*,type(rows,'Row') FROM (SELECT 1 AS id) rows`

		statements := []struct {
			sql  string
			args []any
		}{
			{`INSERT INTO namespaces(namespace_id,owner_id,name,title,status,visibility,mcp_enabled,etag,created_at,updated_at) VALUES(?,'owner',?,?,'active','public',TRUE,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, []any{id, name, name}},
			{`INSERT INTO components(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES(?,?,?,?,'owner','active','shared',?,'reader',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, []any{name, name, name, name, "github.com/viant/datly-studio/dynamic/" + name}},
			{`INSERT INTO component_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES(?,1,'published','dql',?,?,'{}','1','hash','{}','valid','v1','v1',1,'owner',CURRENT_TIMESTAMP)`, []any{name, dql, dql}},
			{`INSERT INTO component_publications(report_id,active_version_no,active_generation,desired_generation,publication_status,runtime_revision,spec_hash,published_by,published_at) VALUES(?,1,1,1,'active','fixture','hash','owner',CURRENT_TIMESTAMP)`, []any{name}},
		}
		for _, statement := range statements {
			if _, err := db.Exec(statement.sql, statement.args...); err != nil {
				t.Fatal(err)
			}
		}

	}
	// A resource-only namespace has no connector or component dependency.
	ids["guides"] = namespaceaccess.ID("owner", "guides")
	if _, err = db.Exec(`INSERT INTO namespaces(namespace_id,owner_id,name,title,status,visibility,mcp_enabled,etag,created_at,updated_at) VALUES(?,'owner','guides','Guides','active','public',TRUE,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, ids["guides"]); err != nil {
		t.Fatal(err)
	}
	independentSkill, err := namespaceskill.Compile(ctx, ids["guides"], "guide", 1, namespaceskill.Bundle{Content: "---\nname: guide\ndescription: Independent namespace guide\n---\nPublished instructions.", Files: []namespaceskill.File{{Path: "references/query.md", Content: "Use scoped MCP tools."}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO namespace_skills(namespace_id,skill_id,name,title,description,latest_revision,published_revision,status,etag,created_at,created_by,updated_at,updated_by) VALUES(?,'guide','guide','Draft title','Draft description',1,1,'active',2,CURRENT_TIMESTAMP,'owner',CURRENT_TIMESTAMP,'owner')`, ids["guides"]); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO namespace_skill_revisions(namespace_id,skill_id,revision,bundle_json,content_sha256,created_at,created_by,updated_at,updated_by) VALUES(?,'guide',1,?,?,CURRENT_TIMESTAMP,'owner',CURRENT_TIMESTAMP,'owner')`, ids["guides"], string(independentSkill.JSON), independentSkill.SHA256); err != nil {
		t.Fatal(err)
	}
	moduleRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	config := Config{HTTP: Listener{Address: "127.0.0.1:0"}, MCP: Listener{Address: "127.0.0.1:0"}, Authentication: Authentication{DefaultMode: "public"}, Studio: Studio{Driver: "sqlite", DSN: dsn}, Admin: Admin{Token: "fixture-admin"}, RootDir: moduleRoot}
	var manager *namespacemcp.Manager
	if mode == "shared" {
		manager, err = namespacemcp.NewShared(NamespaceFactory(config), "127.0.0.1:0")
	} else {
		manager, err = namespacemcp.New(NamespaceFactory(config))
	}
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := manager.Close(closeCtx); err != nil {
			t.Error(err)
		}
	}()
	reconciler := &namespacemcp.Reconciler{Manager: manager, Definitions: namespacemcp.SQLDefinitions{DB: db}}
	if _, err := reconciler.Apply(ctx); err != nil {
		t.Fatal(err)
	}
	a, _ := manager.Get(ids["alpha"])
	b, _ := manager.Get(ids["beta"])
	callRPC := func(endpoint namespacemcp.Endpoint, method string, params map[string]any) string {
		t.Helper()
		params["_meta"] = map[string]any{"io.modelcontextprotocol/protocolVersion": mcpschema.LatestProtocolVersion, "io.modelcontextprotocol/clientCapabilities": map[string]any{}}
		payload, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
		request, _ := http.NewRequest(http.MethodPost, endpoint.URL(), bytes.NewReader(payload))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json, text/event-stream")
		request.Header.Set("Mcp-Method", method)
		if uri, ok := params["uri"].(string); ok {
			request.Header.Set("Mcp-Name", uri)
		}
		request.Header.Set("Mcp-Protocol-Version", mcpschema.LatestProtocolVersion)
		response, err := (&http.Client{Timeout: 3 * time.Second}).Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil || response.StatusCode != 200 {
			t.Fatalf("namespace MCP status=%d err=%v body=%s", response.StatusCode, err, body)
		}
		return string(body)
	}
	call := func(endpoint namespacemcp.Endpoint) string { return callRPC(endpoint, "tools/list", map[string]any{}) }
	decodeRPC := func(body string) map[string]json.RawMessage {
		t.Helper()
		for _, line := range strings.Split(body, "\n") {
			if strings.HasPrefix(line, "data:") {
				body = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				break
			}
		}
		var result map[string]json.RawMessage
		if err := json.Unmarshal([]byte(body), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	guideEndpoint, exists := manager.Get(ids["guides"])
	if !exists {
		t.Fatal("resource-only namespace listener missing")
	}
	listed := decodeRPC(callRPC(guideEndpoint, "skills/list", map[string]any{}))
	var skills mcpschema.ListSkillsResult
	if err := json.Unmarshal(listed["result"], &skills); err != nil || listed["error"] != nil || len(skills.Skills) != 1 {
		t.Fatalf("namespace skills/list: %s %v", listed, err)
	}
	skillURI := skills.Skills[0].Uri
	got := decodeRPC(callRPC(guideEndpoint, "skills/get", map[string]any{"uri": skillURI}))
	var skill mcpschema.GetSkillResult
	if err := json.Unmarshal(got["result"], &skill); err != nil || got["error"] != nil || skill.Skill.Uri != skillURI || len(skill.Skill.Resources.Files) != 2 || skill.Skill.Frontmatter["description"] != "Independent namespace guide" {
		t.Fatalf("namespace skills/get: %s %v", got, err)
	}
	read := decodeRPC(callRPC(guideEndpoint, "resources/read", map[string]any{"uri": skillURI}))
	var entry mcpschema.ReadResourceResult
	if err := json.Unmarshal(read["result"], &entry); err != nil || len(entry.Contents) != 1 || entry.Contents[0].Text != independentSkill.Bundle.Content {
		t.Fatalf("namespace skill content: %s %v", read, err)
	}
	if foreign := decodeRPC(callRPC(a, "skills/get", map[string]any{"uri": skillURI})); foreign["error"] == nil {
		t.Fatal("foreign namespace skill exposed")
	}
	if _, err := db.Exec(`UPDATE namespace_skills SET published_revision=0 WHERE namespace_id=?`, ids["guides"]); err != nil {
		t.Fatal(err)
	}
	if err := manager.Reload(ctx, ids["guides"], 0); err != nil {
		t.Fatal(err)
	}
	unpublished := decodeRPC(callRPC(guideEndpoint, "skills/list", map[string]any{}))
	// A generation with no skills withdraws the optional skills extension.
	if unpublished["error"] != nil {
		var rpcError struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(unpublished["error"], &rpcError); err != nil || rpcError.Code != -32601 {
			t.Fatalf("unexpected empty skill catalog failure: %s", unpublished)
		}
	} else if err := json.Unmarshal(unpublished["result"], &skills); err != nil || len(skills.Skills) != 0 {
		t.Fatalf("unpublished skill remains visible: %s %v", unpublished, err)
	}
	if removed := decodeRPC(callRPC(guideEndpoint, "resources/read", map[string]any{"uri": skillURI})); removed["error"] == nil {
		t.Fatal("unpublished skill remains readable")
	}
	for name, endpoint := range map[string]namespacemcp.Endpoint{"alpha": a, "beta": b} {
		uri := "mcp://ns/" + name
		response := decodeRPC(callRPC(endpoint, "resources/read", map[string]any{"uri": uri}))
		if response["error"] != nil {
			t.Fatalf("connection resource denied: %s", response["error"])
		}
		var content mcpschema.ReadResourceResult
		if err := json.Unmarshal(response["result"], &content); err != nil || len(content.Contents) != 1 {
			t.Fatalf("connection resource: %+v %v", content, err)
		}
		var client mcpconfig.MCPClient
		if err := json.Unmarshal([]byte(content.Contents[0].Text), &client); err != nil {
			t.Fatal(err)
		}
		if content.Contents[0].Uri != uri || client.Name != name || client.Transport.URL != endpoint.URL() || client.Auth == nil || client.Auth.ShouldPassUserToken() || client.Auth.BackendForFrontend == nil || *client.Auth.BackendForFrontend {
			t.Fatalf("wrong namespace connection: %+v", client)
		}
		other := "alpha"
		if name == "alpha" {
			other = "beta"
		}
		if result := decodeRPC(callRPC(endpoint, "resources/read", map[string]any{"uri": "mcp://ns/" + other})); result["error"] == nil {
			t.Fatal("foreign namespace connection exported")
		}
	}
	aCatalog, bCatalog := call(a), call(b)
	if !strings.Contains(aCatalog, "alpha.read") || strings.Contains(aCatalog, "beta.read") || !strings.Contains(bCatalog, "beta.read") || strings.Contains(bCatalog, "alpha.read") {
		t.Fatal("runtime namespace catalogs crossed")
	}

	admin := (&NamespaceGroup{Listeners: manager}).AdminHandler("fixture-token")
	adminServer := httptest.NewServer(admin)
	defer adminServer.Close()
	adminClient, err := runtimeadmin.New(adminServer.URL, "fixture-token", nil)
	if err != nil {
		t.Fatal(err)
	}
	// The root/admin bridge uses the same MCP server as ordinary clients.
	// Republish the independent skill and prove scoped native list/get/read.
	if _, err := db.Exec(`UPDATE namespace_skills SET published_revision=1 WHERE namespace_id=?`, ids["guides"]); err != nil {
		t.Fatal(err)
	}
	if err := manager.Reload(ctx, ids["guides"], 0); err != nil {
		t.Fatal(err)
	}
	guideContext := sdk.WithNamespaceSelection(ctx, ids["guides"])
	bridgeList, err := adminClient.ReadCatalog(guideContext, "", "skills/list", "", nil)
	if err != nil || json.Unmarshal(bridgeList, &skills) != nil || len(skills.Skills) != 1 || skills.Skills[0].Uri != skillURI {
		t.Fatalf("bridge skill list: %s %v", bridgeList, err)
	}
	bridgeSkill, err := adminClient.ReadCatalog(guideContext, "", "skills/get", skillURI, nil)
	if err != nil || json.Unmarshal(bridgeSkill, &skill) != nil || skill.Skill.Uri != skillURI {
		t.Fatalf("bridge skill get: %s %v", bridgeSkill, err)
	}
	bridgeResource, err := adminClient.ReadCatalog(guideContext, "", "resources/read", skillURI, nil)
	if err != nil || json.Unmarshal(bridgeResource, &entry) != nil || len(entry.Contents) != 1 || entry.Contents[0].Text != independentSkill.Bundle.Content {
		t.Fatalf("bridge resource read: %s %v", bridgeResource, err)
	}
	if _, err := adminClient.ReadCatalog(sdk.WithNamespaceSelection(ctx, ids["alpha"]), "", "skills/get", skillURI, nil); err == nil {
		t.Fatal("bridge exposed another namespace's skill")
	}
	for name, endpoint := range map[string]namespacemcp.Endpoint{"alpha": a, "beta": b} {
		selected := sdk.WithNamespaceSelection(ctx, ids[name])
		status, err := adminClient.ProbeRuntime(selected)
		if err != nil || status.MCPURL != endpoint.URL() {
			t.Fatalf("shared/dedicated status: %+v %v", status, err)
		}
		data, err := adminClient.Connection(selected)
		if err != nil {
			t.Fatal(err)
		}
		var definition mcpconfig.MCPClient
		if err = json.Unmarshal(data, &definition); err != nil || definition.Name != name || definition.Transport.URL != endpoint.URL() {
			t.Fatalf("deployed connection: %s %v", data, err)
		}
	}
	for _, check := range []struct {
		method, query, token string
		status               int
	}{
		{"GET", "namespaceId=" + ids["alpha"], "", 401},
		{"GET", "namespaceId=" + ids["alpha"], "wrong", 401},
		{"POST", "namespaceId=" + ids["alpha"], "fixture-token", 405},
		{"GET", "", "fixture-token", 400},
		{"GET", "namespaceId=" + ids["alpha"] + "&namespaceId=" + ids["beta"], "fixture-token", 400},
		{"GET", "namespaceId=" + strings.Repeat("0", 64), "fixture-token", 404},
	} {
		req := httptest.NewRequest(check.method, "/_studio/connection?"+check.query, nil)
		req.Header.Set("X-Studio-Runtime-Token", check.token)
		response := httptest.NewRecorder()
		admin.ServeHTTP(response, req)
		if response.Code != check.status {
			t.Fatalf("connection guard returned %d want %d", response.Code, check.status)
		}
	}

	for _, check := range []struct {
		token, body string
		status      int
	}{
		{"", `{"namespaceId":"` + ids["alpha"] + `","generation":0}`, 401},
		{"fixture-token", `{"generation":0}`, 400},
		{"fixture-token", `{"namespaceId":"` + strings.Repeat("0", 64) + `","generation":0}`, 404},
		{"fixture-token", `{"namespaceId":"` + ids["alpha"] + `","generation":0}`, 200},
	} {
		request := httptest.NewRequest(http.MethodPost, "/_studio/reload", strings.NewReader(check.body))
		if check.token != "" {
			request.Header.Set("X-Studio-Runtime-Token", check.token)
		}
		response := httptest.NewRecorder()
		admin.ServeHTTP(response, request)
		if response.Code != check.status {
			t.Fatalf("namespace admin status=%d want=%d body=%s", response.Code, check.status, response.Body.String())
		}
	}

	for _, check := range []struct {
		id, token string
		status    int
	}{{ids["alpha"], "", 401}, {"", "fixture-token", 400}, {strings.Repeat("0", 64), "fixture-token", 404}, {ids["alpha"], "fixture-token", 200}, {ids["beta"], "fixture-token", 200}} {
		request := httptest.NewRequest(http.MethodGet, "/_studio/status?namespaceId="+check.id, nil)
		if check.token != "" {
			request.Header.Set("X-Studio-Runtime-Token", check.token)
		}
		response := httptest.NewRecorder()
		admin.ServeHTTP(response, request)
		if response.Code != check.status {
			t.Fatalf("namespace status code=%d want=%d body=%s", response.Code, check.status, response.Body.String())
		}
		if check.status == 200 {
			var status struct {
				NamespaceID string `json:"namespaceId"`
				MCPURL      string `json:"mcpUrl"`
				Revision    int64  `json:"revision"`
			}
			endpoint, _ := manager.Get(check.id)
			if err := json.Unmarshal(response.Body.Bytes(), &status); err != nil || status.NamespaceID != check.id || status.MCPURL != endpoint.URL() || status.Revision < 1 {
				t.Fatalf("namespace status payload=%+v err=%v", status, err)
			}
		}
	}
	if err := manager.Reload(ctx, ids["alpha"], 0); err != nil {
		t.Fatal(err)
	}
	currentA, _ := manager.Get(ids["alpha"])
	currentB, _ := manager.Get(ids["beta"])
	if currentA != a || currentB != b {
		t.Fatal("reload moved listener ports")
	}
	if _, err := db.Exec(`UPDATE namespaces SET visibility='private' WHERE namespace_id=?`, ids["beta"]); err != nil {
		t.Fatal(err)
	}
	if response := decodeRPC(callRPC(b, "resources/read", map[string]any{"uri": "mcp://ns/beta"})); response["error"] == nil {
		t.Fatal("private namespace connection remained public")
	}
	if strings.Contains(call(b), "beta.read") {
		t.Fatal("private namespace catalog remained public")
	}
	if !strings.Contains(call(a), "alpha.read") {
		t.Fatal("namespace B visibility changed A")
	}
}
