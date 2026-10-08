package host

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/runtime/namespacemcp"
	"github.com/viant/datly-studio/schema"
	mcpschema "github.com/viant/mcp-protocol/schema"
	_ "modernc.org/sqlite"
)

func TestNamespaceFactoryBuildsScopedRuntimeCatalogsAndReloads(t *testing.T) {
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
	moduleRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	config := Config{HTTP: Listener{Address: "127.0.0.1:0"}, MCP: Listener{Address: "127.0.0.1:0"}, Authentication: Authentication{DefaultMode: "public"}, Studio: Studio{Driver: "sqlite", DSN: dsn}, Admin: Admin{Token: "fixture-admin"}, RootDir: moduleRoot}
	manager, err := namespacemcp.New(NamespaceFactory(config))
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
	call := func(endpoint namespacemcp.Endpoint) string {
		t.Helper()
		payload, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": map[string]any{"_meta": map[string]any{"io.modelcontextprotocol/protocolVersion": mcpschema.LatestProtocolVersion, "io.modelcontextprotocol/clientCapabilities": map[string]any{}}}})
		request, _ := http.NewRequest(http.MethodPost, endpoint.URL(), bytes.NewReader(payload))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json, text/event-stream")
		request.Header.Set("Mcp-Method", "tools/list")
		request.Header.Set("Mcp-Protocol-Version", mcpschema.LatestProtocolVersion)
		response, err := (&http.Client{Timeout: 3 * time.Second}).Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil || response.StatusCode != 200 {
			t.Fatalf("namespace MCP status=%d err=%v", response.StatusCode, err)
		}
		return string(body)
	}
	aCatalog, bCatalog := call(a), call(b)
	if !strings.Contains(aCatalog, "alpha.read") || strings.Contains(aCatalog, "beta.read") || !strings.Contains(bCatalog, "beta.read") || strings.Contains(bCatalog, "alpha.read") {
		t.Fatal("runtime namespace catalogs crossed")
	}

	admin := (&NamespaceGroup{Listeners: manager}).AdminHandler("fixture-token")
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
	if strings.Contains(call(b), "beta.read") {
		t.Fatal("private namespace catalog remained public")
	}
	if !strings.Contains(call(a), "alpha.read") {
		t.Fatal("namespace B visibility changed A")
	}
}
