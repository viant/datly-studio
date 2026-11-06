package host

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/viant/authz"
	policysql "github.com/viant/authz/component/store/sql"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	primitive "github.com/viant/forge/backend/provider/contract"
	identity "github.com/viant/forge/backend/resource"
	"github.com/viant/jsonrpc/transport/client/http/streamable"
	"github.com/viant/mcp-protocol/schema"
	mcpclient "github.com/viant/mcp/client"
)

func TestStandaloneDatasourceResourceUsesInternalNativeComponent(t *testing.T) {
	f := newStockOAuthFixture(t)
	var token atomic.Value
	token.Store(f.token(t, f.private, "alice", []string{"reader"}, time.Now().Add(time.Minute)))
	uri := "datasource://team/rows"
	ref := ComponentReference{Kind: "dynamic", ID: "records", Revision: "1", Method: "GET", Route: "/records"}
	readRule := authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: "alice"}}
	testDynamicHost(t, true, false, dynamicHostExtension{skipDefaultChecks: true, configure: func(config *Config) {
		config.Access = &ResourceAccessConfig{Tenant: "*", Provider: f.provider}
		config.Forge = &ForgeConfig{ProviderIdentity: "standalone-fixture", DataSources: []ForgeDataSource{{URI: uri, DefinitionPath: "rows.json", Component: ref}}, Policies: []authz.Document{{Resource: authz.Resource{Kind: "datasource", ID: "team/rows", Version: identity.WorkingCandidate, Tenant: "*"}, Revision: 1, Policies: map[string]authz.Policy{"discover": readRule, "describe": readRule, "execute": readRule}}}, Selections: []authz.SelectionDocument{{Resource: authz.ResourceFamily{Kind: "datasource", ID: "team/rows", Tenant: "*"}, Revision: 1, DefaultVersion: identity.WorkingCandidate}}}
		require.NoError(t, os.WriteFile(filepath.Join(config.RootDir, "rows.json"), []byte(`{"schemaVersion":1,"dataSource":{"id":"rows","cardinality":"many","selectors":{"data":"Records"},"parameters":[]}}`), 0600))
		db, err := sql.Open(config.Studio.Driver, config.Studio.DSN)
		require.NoError(t, err)
		defer db.Close()
		var dql string
		require.NoError(t, db.QueryRow("SELECT authored_dql FROM component_versions WHERE report_id='records' AND version_no=1").Scan(&dql))
		dql = strings.Replace(dql, "#setting($_ = $mcp('records.read','Read records'))", "#setting($_ = $internal(true))", 1)
		_, err = db.Exec("UPDATE component_versions SET authored_dql=?,generated_dql=? WHERE report_id='records' AND version_no=1", dql, dql)
		require.NoError(t, err)
	}, verify: func(host *Service, address string) {
		ctx := t.Context()
		rule := authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}
		store, err := authz.NewStaticStore([]authz.Document{{Resource: authz.Resource{Kind: "component", ID: "records", Version: "1", Tenant: "*"}, Revision: 1, Policies: map[string]authz.Policy{"describe": rule, "execute": rule}}})
		require.NoError(t, err)
		host.resourceAccess.Store = store
		require.Empty(t, host.config.Forge.Windows)
		transport, err := streamable.New(ctx, "http://"+address+"/forge/mcp", streamable.WithHTTPClient(&http.Client{Transport: studioBearerTransport{token: &token}}), streamable.WithStateless(), streamable.WithProtocolVersion(schema.LatestProtocolVersion), streamable.WithRequestHeaderProvider(func(_ context.Context, body []byte, headers http.Header) error {
			var request struct {
				Method string
				Params struct{ Name, URI string }
			}
			if err := json.Unmarshal(body, &request); err != nil {
				return err
			}
			headers.Set(schema.HeaderMethod, request.Method)
			if request.Method == schema.MethodToolsCall {
				headers.Set(schema.HeaderName, request.Params.Name)
			}
			return nil
		}))
		require.NoError(t, err)
		client := mcpclient.New("standalone-fixture", "1", transport, mcpclient.WithProtocolVersion(schema.LatestProtocolVersion))
		defer client.Close()
		_, err = client.Initialize(ctx)
		require.NoError(t, err)
		call := func(name string, args map[string]any) (json.RawMessage, bool) {
			result, err := client.CallTool(ctx, &schema.CallToolRequestParams{Name: name, Arguments: args}, mcpclient.WithNoRetry())
			require.NoError(t, err)
			raw, err := json.Marshal(result.StructuredContent)
			require.NoError(t, err)
			return raw, result.IsError != nil && *result.IsError
		}
		tools, err := client.ListTools(ctx, nil)
		require.NoError(t, err)
		for _, tool := range tools.Tools {
			require.NotEqual(t, "records.read", tool.Name)
			require.NotEqual(t, "windows/get", tool.Name)
		}
		_, httpErr := executeRuntimeJSON(oauth.WithBearer(ctx, token.Load().(string)), host.authenticated(host.manager), "GET", "/records", nil)
		require.ErrorContains(t, httpErr, "404")
		listed, failed := call("datasources/list", map[string]any{"namespace": "team"})
		require.False(t, failed)
		require.Contains(t, string(listed), uri)
		gotBytes, failed := call("datasources/get", map[string]any{"uri": uri, "revision": "working"})
		require.False(t, failed)
		var got primitive.GetResult
		require.NoError(t, json.Unmarshal(gotBytes, &got))
		require.NotNil(t, got.ExecutionProof)
		var definition stockDatasourceDefinition
		require.NoError(t, json.Unmarshal(got.Resource.DefinitionBytes, &definition))
		require.Equal(t, "datasources/fetch", definition.DataSource.Backend.Method)
		fetch := func() (json.RawMessage, bool) {
			raw, err := json.Marshal(stockPublishedDatasourceInput{Resource: *got.ResolvedResource, ExecutionProof: got.ExecutionProof, DataSourceID: "rows"})
			require.NoError(t, err)
			var args map[string]any
			require.NoError(t, json.Unmarshal(raw, &args))
			return call("datasources/fetch", args)
		}
		rows, failed := fetch()
		require.False(t, failed)
		require.Contains(t, string(rows), "ready")
		require.Contains(t, string(rows), "primary")
		token.Store(f.token(t, f.private, "bob", []string{"reader"}, time.Now().Add(time.Minute)))
		_, failed = call("datasources/get", map[string]any{"uri": uri, "revision": "working"})
		require.True(t, failed)
		rows, failed = fetch()
		require.True(t, failed, "another actor cannot replay a source proof despite the same component role")
		require.NotContains(t, string(rows), "ready")
		token.Store(f.token(t, f.private, "alice", []string{"reader"}, time.Now().Add(time.Minute)))
		checkStandaloneNamespaceBoundary(ctx, t, host, f, &token, uri)
		_, err = host.studio.ExecContext(ctx, "UPDATE connectors SET status='disabled' WHERE name='source'")
		require.NoError(t, err)
		rows, failed = fetch()
		require.True(t, failed)
		require.NotContains(t, string(rows), "ready")
	}})
}

func checkStandaloneNamespaceBoundary(ctx context.Context, t *testing.T, host *Service, f *stockOAuthFixture, token *atomic.Value, uri string) {
	t.Helper()
	teamID, otherID := namespaceaccess.ID("owner", "team"), namespaceaccess.ID("owner", "other")
	for name, id := range map[string]string{"team": teamID, "other": otherID} {
		_, err := host.studio.ExecContext(ctx, `INSERT INTO namespaces(namespace_id,owner_id,name,title,status,visibility,mcp_enabled,etag,created_at,updated_at)VALUES(?,'owner',?,?,'active','public',TRUE,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, id, name, name)
		require.NoError(t, err)
	}
	_, err := host.studio.ExecContext(ctx, "UPDATE components SET namespace_id=?,namespace='team' WHERE id='records'", teamID)
	require.NoError(t, err)
	policies := &policysql.Store{DB: host.studio}
	rule := authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}
	resource := authz.Resource{Kind: "component", ID: "records", Version: "1", Tenant: "*"}
	previous, err := policies.Get(ctx, resource)
	require.NoError(t, err)
	_, err = policies.Replace(ctx, authz.Document{Resource: resource, Revision: previous.Revision, Policies: map[string]authz.Policy{"describe": rule, "execute": rule}}, previous.Revision, "fixture")
	require.NoError(t, err)
	require.NoError(t, policies.Close(ctx))
	config := host.config
	config.ForgeProvider = nil
	config.MCP.Address = "127.0.0.1:0"
	config.HTTP.Address = "127.0.0.1:0"
	config.NamespaceMCPMode = "shared"
	group, err := NewNamespaceGroup(ctx, config)
	require.NoError(t, err)
	defer group.Close(context.WithoutCancel(ctx))
	changes, err := group.Reconcile(ctx)
	require.NoError(t, err)
	for _, change := range changes {
		require.NotEqual(t, "failed", change.Status, "%+v", change)
	}
	endpoint := func(id string) string {
		return "http://" + group.Listeners.SharedAddress() + "/namespaces/" + id + "/forge/mcp"
	}
	makeClient := func(url string) *mcpclient.Client {
		transport, err := streamable.New(ctx, url, streamable.WithHTTPClient(&http.Client{Transport: studioBearerTransport{token: token}}), streamable.WithStateless(), streamable.WithProtocolVersion(schema.LatestProtocolVersion), streamable.WithRequestHeaderProvider(func(_ context.Context, body []byte, headers http.Header) error {
			var request struct {
				Method string
				Params struct{ Name, Uri string }
			}
			if err := json.Unmarshal(body, &request); err != nil {
				return err
			}
			headers.Set(schema.HeaderMethod, request.Method)
			if request.Method == schema.MethodToolsCall {
				headers.Set(schema.HeaderName, request.Params.Name)
			}
			if request.Method == schema.MethodResourcesRead {
				headers.Set(schema.HeaderName, request.Params.Uri)
			}
			return nil
		}))
		require.NoError(t, err)
		return mcpclient.New("scoped-datasource-fixture", "1", transport, mcpclient.WithProtocolVersion(schema.LatestProtocolVersion))
	}
	allowed := makeClient(endpoint(teamID))
	defer allowed.Close()
	_, err = allowed.Initialize(ctx)
	require.NoError(t, err)
	result, err := allowed.CallTool(ctx, &schema.CallToolRequestParams{Name: "datasources/get", Arguments: map[string]any{"uri": uri, "revision": "working"}}, mcpclient.WithNoRetry())
	require.NoError(t, err)
	require.False(t, result.IsError != nil && *result.IsError)
	raw, err := json.Marshal(result.StructuredContent)
	require.NoError(t, err)
	var got primitive.GetResult
	require.NoError(t, json.Unmarshal(raw, &got))
	require.Equal(t, uri, got.ResolvedResource.URI)
	wrong := makeClient(endpoint(otherID))
	defer wrong.Close()
	_, err = wrong.Initialize(ctx)
	if err == nil {
		_, err = wrong.ReadResource(ctx, &schema.ReadResourceRequestParams{Uri: uri})
	}
	require.Error(t, err, "configured source must not be mounted in another namespace")
	token.Store(f.token(t, f.private, "bob", []string{"reader"}, time.Now().Add(time.Minute)))
	denied, err := allowed.CallTool(ctx, &schema.CallToolRequestParams{Name: "datasources/get", Arguments: map[string]any{"uri": uri, "revision": "working"}}, mcpclient.WithNoRetry())
	require.True(t, err != nil || denied.IsError != nil && *denied.IsError)
	token.Store(f.token(t, f.private, "alice", []string{"reader"}, time.Now().Add(time.Minute)))
	_, err = host.studio.ExecContext(ctx, "UPDATE namespaces SET mcp_enabled=FALSE WHERE namespace_id=?", teamID)
	require.NoError(t, err)
	_, err = group.Reconcile(ctx)
	require.NoError(t, err)
	_, err = allowed.ReadResource(ctx, &schema.ReadResourceRequestParams{Uri: uri})
	require.Error(t, err, "disabled namespace lifecycle must remove the datasource endpoint")
}
