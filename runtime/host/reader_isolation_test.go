package host

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/schema"
	studiohost "github.com/viant/datly-studio/studio/host"
	"github.com/viant/datly/transcribe"
)

func TestNamespaceColdStartIsolatesMissingReaderSource(t *testing.T) {
	for _, failure := range []string{"missing_source", "invalid_dql", "invalid_resource", "broken_connector"} {
		t.Run(failure, func(t *testing.T) { testNamespaceReaderIsolation(t, failure) })
	}
}
func testNamespaceReaderIsolation(t *testing.T, failure string) {
	ctx := t.Context()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/isolation\n\ngo 1.25.0\n"), 0600))
	dsn := "file:" + filepath.Join(root, "studio.db")
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, schema.ApplySQLite(ctx, db, "studio"))
	ns := namespaceaccess.ID("owner", "isolated")
	_, err = db.Exec(`CREATE TABLE records(id INTEGER PRIMARY KEY,name TEXT); INSERT INTO records VALUES(1,'healthy');`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO connectors(name,driver,dsn_template,owner_id,status,etag,created_at,updated_at) VALUES('source','sqlite',?,'owner','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);`, dsn)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO namespaces(namespace_id,owner_id,name,title,status,visibility,mcp_enabled,etag,created_at,updated_at) VALUES(?,'owner','isolated','Isolated','active','public',TRUE,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);`, ns)
	require.NoError(t, err)
	var repairedDQL string
	if failure == "broken_connector" {
		_, err = db.Exec(`INSERT INTO connectors(name,driver,dsn_template,owner_id,status,etag,created_at,updated_at) VALUES('broken_source','unlinked-driver',?,'owner','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, dsn)
		require.NoError(t, err)
	}
	for _, id := range []string{"healthy", "broken"} {
		table := "records"
		if id == "broken" {
			table = "removed_source"
		}
		dql := fmt.Sprintf(`#package('example.com/isolation/%s')
#setting($_ = $connector('source'))
#setting($_ = $route('/%s','GET'))
#setting($_ = $mcp('%s','Fixture reader'))
#define($_ = $Rows<[]*Row>(output/view))
SELECT r.*,type(r,'Row') FROM (SELECT id,name FROM %s) r`, id, id, id, table)
		if id == "broken" {
			if failure == "broken_connector" {
				dql = strings.Replace(dql, "$connector('source')", "$connector('broken_source')", 1)
			}
			repairedDQL = dql
			if failure == "invalid_dql" {
				dql = "not a valid DQL component"
			}
		}
		_, err = db.Exec(`INSERT INTO components(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES(?,'isolated',?,?,'owner','active','source',?,'reader',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);`, id, id, id, "example.com/isolation/"+id)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO component_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES(?,1,'published','dql',?,?,'{}','1','hash','{}','valid','v1','v1',1,'owner',CURRENT_TIMESTAMP);`, id, dql, dql)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO component_publications(report_id,active_version_no,active_generation,desired_generation,publication_status,runtime_revision,spec_hash,published_by,published_at) VALUES(?,1,1,1,'active','fixture','hash','owner',CURRENT_TIMESTAMP);`, id)
		require.NoError(t, err)
	}
	if failure == "invalid_resource" {
		_, err = db.Exec(`INSERT INTO component_resource_files(report_id,version_no,resource_id,namespace,resource_path,content,content_size,content_sha256,is_binary,created_at) VALUES('broken',1,'invalid','fixture','../escape','unsafe',6,'fixture',0,CURRENT_TIMESTAMP)`)
		require.NoError(t, err)
	}
	config := Config{NamespaceID: ns, HTTP: Listener{Address: "127.0.0.1:0"}, MCP: Listener{Address: "127.0.0.1:0"}, Authentication: Authentication{DefaultMode: "public"}, Studio: Studio{Driver: "sqlite", DSN: dsn}, Admin: Admin{Token: "fixture"}, RootDir: root}
	service, err := New(ctx, config)
	require.NoError(t, err)
	var callsMu sync.Mutex
	calls := map[string]int{}
	service.compileRuntimeContract = func(ctx context.Context, source *transcribe.Source) (*transcribe.RuntimeContract, error) {
		callsMu.Lock()
		calls[source.Scope]++
		callsMu.Unlock()
		return transcribe.NewCompiler().RuntimeContracts(ctx, root, source)
	}
	t.Cleanup(func() { _ = service.Close(context.Background()) })
	require.NoError(t, service.Start(ctx), "one missing source must not prevent namespace startup")
	address, _ := service.Addresses()
	response, err := http.Get("http://" + address + "/healthy")
	require.NoError(t, err)
	data, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode)
	require.Contains(t, string(data), "healthy")
	response, err = http.Get("http://" + address + "/broken")
	require.NoError(t, err)
	_ = response.Body.Close()
	require.GreaterOrEqual(t, response.StatusCode, 400)
	require.NotEmpty(t, service.RuntimeDiagnostics())
	require.Equal(t, "broken", service.RuntimeDiagnostics()[0].Field)
	statusRequest, e := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+address+"/_studio/status", nil)
	require.NoError(t, e)
	statusRequest.Header.Set("X-Studio-Runtime-Token", "fixture")
	statusResponse, e := http.DefaultClient.Do(statusRequest)
	require.NoError(t, e)
	var status struct {
		Status      string
		Diagnostics []struct{ Field, Stage, Message string }
	}
	require.NoError(t, json.NewDecoder(statusResponse.Body).Decode(&status))
	_ = statusResponse.Body.Close()
	require.Equal(t, "ready", status.Status)
	require.NotEmpty(t, status.Diagnostics)
	require.Equal(t, "broken", status.Diagnostics[0].Field)
	require.NotEmpty(t, status.Diagnostics[0].Stage)
	require.NotContains(t, status.Diagnostics[0].Message, "SELECT")
	_, mcpAddress := service.Addresses()
	call := func(method string, params map[string]any) map[string]any {
		payload, e := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
		require.NoError(t, e)
		req, e := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+mcpAddress+"/mcp", bytes.NewReader(payload))
		require.NoError(t, e)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("Mcp-Method", method)
		resp, e := http.DefaultClient.Do(req)
		require.NoError(t, e)
		defer resp.Body.Close()
		var result map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		return result
	}
	listed := call("tools/list", map[string]any{})
	encoded, _ := json.Marshal(listed)
	require.Contains(t, string(encoded), `"name":"healthy"`)
	require.NotContains(t, string(encoded), `"name":"broken"`)
	result := call("tools/call", map[string]any{"name": "healthy", "arguments": map[string]any{}})
	require.Nil(t, result["error"])
	wire, ok := result["result"].(map[string]any)
	require.True(t, ok)
	require.NotEqual(t, true, wire["isError"], "healthy tool must execute, not merely return a tool error mentioning its name")
	encoded, _ = json.Marshal(result)
	require.Contains(t, string(encoded), "healthy")
	callsMu.Lock()
	require.Equal(t, 1, calls["example.com/isolation/healthy"], "a sibling failure must not recompile healthy contracts")
	callsMu.Unlock()
	definitions, e := service.definitions(ctx, nil)
	require.NoError(t, e)
	seed, e := (studiohost.Config{}).RuntimeTypes()
	require.NoError(t, e)
	_, e = service.compileDefinitions(ctx, seed, definitions)
	require.Error(t, e, "exact version validation remains strict")
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	before := service.manager.Revision()
	require.ErrorIs(t, service.Reload(canceled, 0), context.Canceled)
	require.Equal(t, before, service.manager.Revision(), "cancellation must not install a partial generation")

	_, err = db.Exec(`CREATE TABLE removed_source(id INTEGER PRIMARY KEY,name TEXT);INSERT INTO removed_source VALUES(2,'repaired')`)
	require.NoError(t, err)
	if failure == "broken_connector" {
		_, err = db.Exec(`UPDATE connectors SET driver='sqlite',etag=2 WHERE name='broken_source'`)
		require.NoError(t, err)
	}
	if failure == "invalid_resource" {
		_, err = db.Exec(`DELETE FROM component_resource_files WHERE report_id='broken' AND resource_id='invalid'`)
		require.NoError(t, err)
	}
	if failure == "invalid_dql" {
		_, err = db.Exec(`UPDATE component_versions SET authored_dql=?,generated_dql=? WHERE report_id='broken'`, repairedDQL, repairedDQL)
		require.NoError(t, err)
	}
	require.NoError(t, service.Reload(ctx, 0))
	require.Empty(t, service.RuntimeDiagnostics())
	response, err = http.Get("http://" + address + "/broken")
	require.NoError(t, err)
	data, err = io.ReadAll(response.Body)
	_ = response.Body.Close()
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode)
	require.Contains(t, string(data), "repaired")
	callsMu.Lock()
	require.Equal(t, 1, calls["example.com/isolation/healthy"], "repair reuses unchanged healthy compiled artifact")
	callsMu.Unlock()
}
