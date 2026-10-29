package studiohost

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	runtimehost "github.com/viant/datly-studio/runtime/host"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/studio/predicatecatalog/testdata/extension"
	mcpschema "github.com/viant/mcp-protocol/schema"
)

// A replacement for a retired linked predicate must be loadable even when the
// active publication cannot compile in this binary. Staging remains SDK-owned;
// these fixtures exercise the runtime callback after that authorized stage.
func TestColdPublicationLoadsRequestedGenerationWithoutCompilingRetiredHandler(t *testing.T) {
	_ = extension.LinkedType
	t.Setenv("STUDIO_PREDICATE_PACKAGES", "github.com/viant/datly-studio/studio/predicatecatalog/testdata/extension")
	for _, mode := range []string{"shared", "separate"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			dsn := "file:" + filepath.Join(t.TempDir(), "studio.db")
			db, err := sql.Open("sqlite", dsn)
			require.NoError(t, err)
			defer db.Close()
			require.NoError(t, schema.ApplySQLite(ctx, db, "studio"))
			id, foreign := strings.Repeat("a", 64), strings.Repeat("b", 64)
			_, err = db.ExecContext(ctx, `CREATE TABLE records(id INTEGER,project_id INTEGER,name TEXT);
INSERT INTO records VALUES(1,101,'first');
INSERT INTO connectors(name,driver,dsn_template,owner_id,status,etag,created_at,updated_at) VALUES('source','sqlite',?,'owner','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);`, dsn)
			require.NoError(t, err)
			_, err = db.ExecContext(ctx, `INSERT INTO namespaces(namespace_id,name,title,owner_id,status,visibility,mcp_enabled,etag,created_at,updated_at) VALUES(?,'test','Test','owner','active','public',1,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO components(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES('records','test','records','Records','owner','active','source','github.com/viant/datly-studio/dynamic/records','reader',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);`, id)
			require.NoError(t, err)
			current := `#package('github.com/viant/datly-studio/dynamic/records')
#import('extension','github.com/viant/datly-studio/studio/predicatecatalog/testdata/extension')
#setting($_ = $connector('source'))
#setting($_ = $route('/records','GET'))
#setting($_ = $mcp('records.list','List records'))
#define($_ = $Selected<bool>(query/selected).WithTag('json:"selected"').Required().WithPredicate(0,'handler','extension.ReaderScope'))
#define($_ = $Records<[]*Record>(output/view))
SELECT records.*,type(records,'Record') FROM (SELECT t.id,t.project_id,t.name FROM records t ${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("WHERE")}) records`
			old := strings.ReplaceAll(current, "extension.ReaderScope", "extension.RetiredScope")
			for version, dql := range []string{old, current} {
				_, err = db.ExecContext(ctx, `INSERT INTO component_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES('records',?,'validated','dql',?,?,'{}','1',?,'{}','valid','v1','v1',?,'owner',CURRENT_TIMESTAMP)`, version+1, dql, dql, fmt.Sprint(version+1), version+1)
				require.NoError(t, err)
			}
			for _, row := range []struct {
				generation        int
				namespace, status string
			}{{1, id, "active"}, {2, id, "building"}, {3, id, "failed"}, {4, id, "retired"}, {5, foreign, "building"}} {
				_, err = db.ExecContext(ctx, `INSERT INTO runtime_generations(namespace_id,generation_no,source_revision,status,report_count,build_manifest_json,requested_by,requested_at) VALUES(?,?,?,?,1,'{}','owner',CURRENT_TIMESTAMP)`, row.namespace, row.generation, fmt.Sprint(row.generation), row.status)
				require.NoError(t, err)
			}
			_, err = db.ExecContext(ctx, `INSERT INTO component_publications(report_id,active_version_no,desired_version_no,active_generation,desired_generation,publication_status,runtime_revision,spec_hash,published_by,published_at) VALUES('records',1,2,1,2,'pending','candidate','2','owner',CURRENT_TIMESTAMP)`)
			require.NoError(t, err)
			root, err := filepath.Abs("../..")
			require.NoError(t, err)
			config := runtimehost.Config{InProcess: true, RootDir: root, NamespaceMCPMode: mode, HTTP: runtimehost.Listener{Address: "127.0.0.1:0"}, MCP: runtimehost.Listener{Address: "127.0.0.1:0"}, Authentication: runtimehost.Authentication{DefaultMode: "public"}, Studio: runtimehost.Studio{Driver: "sqlite", DSN: dsn}}
			group, err := runtimehost.NewNamespaceGroup(ctx, config)
			require.NoError(t, err)
			defer group.Close(ctx)
			selected := sdk.WithNamespaceSelection(ctx, id)
			var active atomic.Pointer[runtimehost.NamespaceGroup]
			active.Store(group)
			control := inProcessControl(&active)
			for _, generation := range []int64{-1, 99, 3, 4, 5} {
				require.Error(t, control.Reload(selected, generation), "unavailable generation %d", generation)
				_, exists := group.Listeners.Get(id)
				require.False(t, exists, "failed candidate must not install a cold listener")
			}
			_, err = group.Listeners.Rebind(ctx, id, 0)
			require.ErrorContains(t, err, "RetiredScope", "ordinary startup must continue loading active generation")
			require.NoError(t, control.Reload(selected, 2), "cold publication must load the replacement directly")
			endpoint, exists := group.Listeners.Get(id)
			require.True(t, exists)
			call := func(target, method string) string {
				arguments := ""
				if method == "tools/call" {
					arguments = `,"name":"records.list","arguments":{"selected":true}`
				}
				request, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"`+method+`","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"`+mcpschema.LatestProtocolVersion+`","io.modelcontextprotocol/clientCapabilities":{}}`+arguments+`}}`))
				require.NoError(t, err)
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Accept", "application/json, text/event-stream")
				request.Header.Set("Mcp-Method", method)
				if method == "tools/call" {
					request.Header.Set("Mcp-Name", "records.list")
				}
				request.Header.Set("Mcp-Protocol-Version", mcpschema.LatestProtocolVersion)
				response, err := (&http.Client{Timeout: 3 * time.Second}).Do(request)
				require.NoError(t, err)
				defer response.Body.Close()
				body, err := io.ReadAll(response.Body)
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, response.StatusCode)
				return string(body)
			}
			require.Contains(t, call(endpoint.URL(), "tools/list"), "records.list")
			result := call(endpoint.URL(), "tools/call")
			require.Contains(t, result, "first", "the current linked predicate must execute against fresh rows")
			require.NotContains(t, result, `"isError":true`)
			// A later invalid staged target retains the live build. Reloading its active
			// generation is the same primitive used by SDK activation compensation.
			_, err = db.ExecContext(ctx, `UPDATE runtime_generations SET status='active' WHERE generation_no=2;
UPDATE runtime_generations SET status='building' WHERE generation_no=3;
UPDATE component_publications SET active_version_no=2,active_generation=2,desired_version_no=1,desired_generation=3 WHERE report_id='records'`)
			require.NoError(t, err)
			require.ErrorContains(t, control.Reload(selected, 3), "RetiredScope")
			retained, exists := group.Listeners.Get(id)
			require.True(t, exists)
			require.Equal(t, endpoint, retained)
			require.Contains(t, call(retained.URL(), "tools/list"), "records.list")
			require.NoError(t, control.Reload(selected, 2), "active generation compensation must remain supported")
			restored, exists := group.Listeners.Get(id)
			require.True(t, exists)
			require.Contains(t, call(restored.URL(), "tools/list"), "records.list")
			var status string
			require.NoError(t, db.QueryRowContext(ctx, `SELECT publication_status FROM component_publications WHERE report_id='records'`).Scan(&status))
			require.Equal(t, "pending", status, "runtime reload must not activate SDK publication state")
		})
	}
}
