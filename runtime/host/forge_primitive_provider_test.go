package host

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	dsproto "github.com/viant/agently-core/protocol/datasource"
	mcpcfg "github.com/viant/agently-core/protocol/mcp/config"
	manager "github.com/viant/agently-core/protocol/mcp/manager"
	primitive "github.com/viant/agently-core/protocol/primitive"
	identity "github.com/viant/agently-core/protocol/resource"
	windowprotocol "github.com/viant/agently-core/protocol/window"
	requestctx "github.com/viant/agently-core/runtime/requestctx"
	dssvc "github.com/viant/agently-core/service/datasource"
	bridge "github.com/viant/agently-core/service/primitiveprovider"
	coreresource "github.com/viant/agently-core/service/resource"
	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/forge/backend/types"
	"github.com/viant/jsonrpc/transport/client/http/streamable"
	"github.com/viant/mcp-protocol/schema"
	mcpclient "github.com/viant/mcp/client"
)

type studioGatewayOptions map[string]*mcpcfg.MCPClient

func (o studioGatewayOptions) Options(_ context.Context, name string) (*mcpcfg.MCPClient, error) {
	return o[name], nil
}
func (o studioGatewayOptions) Names(context.Context) ([]string, error) {
	return []string{"studio"}, nil
}

type studioBearerTransport struct{ token *atomic.Value }

func (t studioBearerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	copy := request.Clone(request.Context())
	copy.Header = request.Header.Clone()
	copy.Header.Set("Authorization", "Bearer "+t.token.Load().(string))
	return http.DefaultTransport.RoundTrip(copy)
}

func TestConfiguredStudioWindowsThroughCommonGatewayAndNativeMCP(t *testing.T) {
	for _, count := range []int{1, 20} {
		t.Run(fmt.Sprint(count), func(t *testing.T) { testConfiguredStudioGateway(t, count) })
	}
}

func testConfiguredStudioGateway(t *testing.T, count int) {
	f := newStockOAuthFixture(t)
	var token atomic.Value
	token.Store(f.token(t, f.private, "alice", []string{"reader"}, time.Now().Add(time.Minute)))
	uri := "window://team/overview"
	ref := ComponentReference{Kind: "dynamic", ID: "records", Revision: "1", Method: http.MethodGet, Route: "/records"}
	window := []byte("window:\n  view:\n    content:\n      id: original-root\n      title: Studio configured window\ndataSources:\n  records:\n    id: records\n    selectors:\n      data: Records\n    cardinality: many\n    parameters: []\n")
	policy := authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}
	var sourceFile string
	testDynamicHost(t, true, false, dynamicHostExtension{configure: func(config *Config) {
		config.Access = &ResourceAccessConfig{Tenant: "*", Provider: f.provider}
		config.Forge = &ForgeConfig{ProviderIdentity: "studio-fixture", Windows: []ForgeWindow{{URI: uri, DefinitionPath: "original-window.yaml", Components: map[string]ComponentReference{"records": ref}}},
			Policies:   []authz.Document{{Resource: authz.Resource{Kind: "window", ID: "team/overview", Version: identity.WorkingCandidate, Tenant: "*"}, Revision: 1, Policies: map[string]authz.Policy{"discover": policy, "describe": policy, "execute": policy}}},
			Selections: []authz.SelectionDocument{{Resource: authz.ResourceFamily{Kind: "window", ID: "team/overview", Tenant: "*"}, Revision: 1, DefaultVersion: identity.WorkingCandidate}}}
		for i := 1; i < count; i++ {
			name := fmt.Sprintf("overview-%02d", i)
			config.Forge.Windows = append(config.Forge.Windows, ForgeWindow{URI: "window://team/" + name, DefinitionPath: "original-window.yaml", Components: map[string]ComponentReference{"records": ref}})
			document := configWindowPolicy(policy)
			document.Resource.ID = "team/" + name
			config.Forge.Policies = append(config.Forge.Policies, document)
			config.Forge.Selections = append(config.Forge.Selections, authz.SelectionDocument{Resource: authz.ResourceFamily{Kind: "window", ID: "team/" + name, Tenant: "*"}, Revision: 1, DefaultVersion: identity.WorkingCandidate})
		}
		sourceFile = filepath.Join(config.RootDir, "original-window.yaml")
		require.NoError(t, os.WriteFile(sourceFile, window, 0600))
	}, verify: func(host *Service, address string) {
		// Native component authority remains an independent gate.
		store, err := authz.NewStaticStore([]authz.Document{configWindowPolicy(policy), {Resource: authz.Resource{Kind: "component", ID: "records", Version: "1", Tenant: "*"}, Revision: 1, Policies: map[string]authz.Policy{"describe": {Mode: "public"}, "execute": {Mode: "public"}}}})
		require.NoError(t, err)
		host.resourceAccess.Store = store
		ctx := context.Background()
		transport, err := streamable.New(ctx, "http://"+address+"/forge/mcp", streamable.WithHTTPClient(&http.Client{Transport: studioBearerTransport{token: &token}}), streamable.WithStateless(), streamable.WithProtocolVersion(schema.LatestProtocolVersion), streamable.WithRequestHeaderProvider(func(_ context.Context, body []byte, headers http.Header) error {
			var request struct {
				Method string
				Params struct{ Name string }
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
		client := mcpclient.New("studio-parity", "1", transport, mcpclient.WithProtocolVersion(schema.LatestProtocolVersion))
		defer client.Close()
		_, err = client.Initialize(ctx)
		require.NoError(t, err)
		tools, err := client.ListTools(ctx, nil)
		require.NoError(t, err)
		installed := map[string]bool{}
		for _, tool := range tools.Tools {
			installed[tool.Name] = true
		}
		for _, name := range []string{"namespaces/list", "namespaces/get", "windows/list", "windows/get", "windows/datasource", windowprotocol.CatalogTool, windowprotocol.DefinitionTool, windowprotocol.FetchTool} {
			require.True(t, installed[name], name)
		}
		mgr, err := manager.New(studioGatewayOptions{"studio": {}}, manager.WithClientFactory(func(context.Context, string, string) (mcpclient.Interface, error) { return client, nil }))
		require.NoError(t, err)
		defer mgr.CloseConversation("")
		actor := func(ctx context.Context) (identity.VerifiedActor, error) {
			p, err := f.provider.ResolvePrincipal(oauth.WithBearer(ctx, token.Load().(string)))
			if err != nil {
				return identity.VerifiedActor{}, err
			}
			return identity.VerifiedActor{Subject: p.Facts.Subject, Issuer: p.Facts.Issuer, TenantID: p.Facts.Tenant, AccountID: p.AccountID, IdentityRevision: p.IdentityRevision, ValidUntil: p.Facts.ValidUntil}, nil
		}
		gateway := coreresource.NewGateway(mgr, actor, func(ctx context.Context, original identity.VerifiedActor) error {
			current, err := actor(ctx)
			if err != nil || original.Subject != current.Subject || original.IdentityRevision != current.IdentityRevision {
				return identity.ErrResourceDenied
			}
			return nil
		}, "gateway-not-provider")
		defer gateway.Close()
		listed, err := gateway.List(ctx, "window", "team")
		require.NoError(t, err)
		require.Len(t, listed, count)
		require.Equal(t, uri, listed[0].Resource.URI)
		require.Equal(t, "studio-fixture", listed[0].Connection.ProviderIdentity)
		got, err := gateway.Get(ctx, listed[0].Connection, identity.ResourceRef{URI: uri}, nil)
		require.NoError(t, err)
		require.NotNil(t, got.ExecutionProof, "provider proof must survive common Gateway transport")
		selected, err := types.SelectWindowResource(got.Resource.DefinitionBytes, nil)
		require.NoError(t, err)
		require.Equal(t, "original-root", selected.Window.View.Content.ID)
		require.Equal(t, "Studio configured window", selected.Window.View.Content.Title)
		require.Equal(t, "team", selected.Window.Namespace)
		require.Equal(t, "Records", selected.Window.DataSource["records"].Selectors.Data)
		var descriptor windowprotocol.DataSource
		require.NoError(t, json.Unmarshal(selected.DataSources["records"], &descriptor))
		require.Equal(t, "provider", descriptor.Backend.Ownership)
		require.Equal(t, "1", descriptor.Backend.Component.Revision)
		// Exercise the normal consumer datasource service, not only a direct
		// provider callback: both original provider and host proofs cross MCP.
		hostProof, err := types.NewWindowTargetHMAC([]byte(strings.Repeat("h", 32)))
		require.NoError(t, err)
		catalog := &coreresource.WindowCatalog{Gateway: gateway, TargetProof: hostProof, Admission: func(ctx context.Context, _ identity.ResolvedResource, _ *types.Window) error {
			_, err := actor(ctx)
			return err
		}}
		opened, err := catalog.Get(ctx, &bridge.WindowDefinitionGetInput{WindowID: uri})
		require.NoError(t, err)
		uiBridge := bridge.NewService(&bridge.Config{WindowDefinitions: catalog, Token: "synthetic-ui-token", RequireToken: true, LocalOnly: true, ResolvedWindowAuthorizer: func(ctx context.Context, _ identity.ResolvedResource, _ string, _ map[string]any) error {
			_, err := actor(ctx)
			return err
		}})
		uiServer := httptest.NewServer(http.HandlerFunc(uiBridge.Hub().ServeWS))
		defer uiServer.Close()
		uiClient, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(uiServer.URL, "http"), nil)
		require.NoError(t, err)
		defer uiClient.Close()
		require.NoError(t, uiClient.WriteJSON(map[string]any{"type": "ui.hello", "clientId": "studio-renderer", "token": "synthetic-ui-token"}))
		worker := make(chan error, 1)
		go func() {
			var command struct {
				ID, Method string
				Params     map[string]json.RawMessage
			}
			if err := uiClient.ReadJSON(&command); err != nil {
				worker <- err
				return
			}
			if command.Method != "ui.window.open" {
				worker <- fmt.Errorf("unexpected UI method %s", command.Method)
				return
			}
			raw, _ := json.Marshal(command.Params)
			if !strings.Contains(string(raw), "original-root") || !strings.Contains(string(raw), "executionProof") || strings.Contains(string(raw), "forged-metadata") {
				worker <- fmt.Errorf("UI delivery lost exact approved metadata/proofs")
				return
			}
			worker <- uiClient.WriteJSON(map[string]any{"id": command.ID, "ok": true, "result": map[string]any{"windowId": "studio-instance"}})
		}()
		// Registration is an actual frontend hello, not a fabricated catalog.
		require.Eventually(t, func() bool { return len(uiBridge.Hub().ListClients("default")) == 1 }, time.Second, time.Millisecond)
		delivered, err := uiBridge.UICommand(ctx, &bridge.UICommandInput{ClientID: "studio-renderer", Method: "ui.window.open", Params: map[string]any{"windowKey": uri, "options": map[string]any{"inlineMetadata": map[string]any{"view": "forged-metadata"}}}})
		require.NoError(t, err)
		require.True(t, delivered.OK)
		require.NoError(t, <-worker)
		fetcher := dssvc.New(dssvc.Options{ResolveResource: func(ctx context.Context, pin identity.ResolvedResource) (*identity.ResolvedResource, error) {
			resolved, err := gateway.Get(ctx, listed[0].Connection, identity.ResourceRef{URI: pin.URI}, &pin)
			if err != nil {
				return nil, err
			}
			return resolved.ResolvedResource, nil
		}, ResolveDefinition: catalog.ResolveDatasource, AuthorizeDefinition: func(ctx context.Context, _ *dsproto.DataSource, _ map[string]interface{}) error {
			_, err := actor(ctx)
			return err
		}, ProviderExecute: func(ctx context.Context, ds *dsproto.DataSource, args map[string]interface{}) (json.RawMessage, error) {
			pin, _ := requestctx.ResolvedResourceFromContext(ctx)
			target, _ := requestctx.WindowTargetFromContext(ctx)
			return catalog.FetchProviderDatasource(ctx, *pin, target, ds, args)
		}})
		projected, err := fetcher.Fetch(ctx, "records", nil, dssvc.FetchOptions{Resource: opened.Definition.Resource, Target: opened.Definition.ResourceTarget})
		require.NoError(t, err)
		require.Len(t, projected.Rows, 1)
		require.Equal(t, "ready", projected.Rows[0]["Name"])
		callFetch := func(pin identity.ResolvedResource) *schema.CallToolResult {
			raw, _ := json.Marshal(stockDatasourceInput{Resource: pin, DataSourceID: "records", Inputs: map[string]any{}, ExecutionProof: got.ExecutionProof})
			var args map[string]any
			require.NoError(t, json.Unmarshal(raw, &args))
			result, err := client.CallTool(ctx, &schema.CallToolRequestParams{Name: "windows/datasource", Arguments: args}, mcpclient.WithNoRetry())
			require.NoError(t, err)
			return result
		}
		fetched := callFetch(*got.ResolvedResource)
		require.True(t, fetched.IsError == nil || !*fetched.IsError)
		body, _ := json.Marshal(fetched.StructuredContent)
		require.Contains(t, string(body), "primary")
		deadlineTamper := *got.ResolvedResource
		deadlineTamper.ValidUntil = got.ExecutionProof.Resource.ValidUntil.Add(time.Second)
		require.True(t, *callFetch(deadlineTamper).IsError, "changing only ValidUntil cannot extend the original provider lease")
		narrowed := *got.ResolvedResource
		narrowed.ValidUntil = narrowed.ValidUntil.Add(-time.Second)
		require.True(t, !*callFetch(narrowed).IsError, "a trusted host may retain an independently narrower lease")
		for _, mutate := range []func(*identity.ResolvedResource){func(p *identity.ResolvedResource) { p.ProviderIdentity = "other" }, func(p *identity.ResolvedResource) { p.URI = "window://team/other" }, func(p *identity.ResolvedResource) { p.AuthorityBinding = "other-actor" }} {
			altered := *got.ResolvedResource
			mutate(&altered)
			require.True(t, *callFetch(altered).IsError)
		}
		originalProof := got.ExecutionProof
		proofCopy := *originalProof
		proofCopy.Binding += "changed-variant"
		got.ExecutionProof = &proofCopy
		require.True(t, *callFetch(*got.ResolvedResource).IsError)
		got.ExecutionProof = originalProof
		originalSigner := host.windowExecutionProof
		restarted, err := types.NewWindowTargetHMAC([]byte(strings.Repeat("x", 32)))
		require.NoError(t, err)
		host.windowExecutionProof = restarted
		require.True(t, *callFetch(*got.ResolvedResource).IsError, "another process key cannot accept old execution proof")
		host.windowExecutionProof = originalSigner
		// Shared read plans must not retain the prior verified caller.
		aliceToken := token.Load().(string)
		bobToken := f.token(t, f.private, "bob", []string{"reader"}, time.Now().Add(time.Minute))
		token.Store(bobToken)
		bob, err := gateway.Get(ctx, listed[0].Connection, identity.ResourceRef{URI: uri}, nil)
		require.NoError(t, err)
		require.NotEqual(t, got.ResolvedResource.AuthorityBinding, bob.ResolvedResource.AuthorityBinding)
		require.True(t, *callFetch(*got.ResolvedResource).IsError, "another authorized actor cannot reuse the original authority pin")
		token.Store(aliceToken)
		var group sync.WaitGroup
		errors := make(chan error, 8)
		for i := 0; i < 8; i++ {
			group.Add(1)
			credential := aliceToken
			if i%2 == 1 {
				credential = bobToken
			}
			go func() {
				defer group.Done()
				_, err := host.windowPrimitives.Get(oauth.WithBearer(context.Background(), credential), "window", primitive.GetRequest{URI: uri})
				errors <- err
			}()
		}
		group.Wait()
		close(errors)
		for err := range errors {
			require.NoError(t, err)
		}
		expired := *got.ResolvedResource
		expired.ValidUntil = time.Now().Add(-time.Second)
		require.True(t, *callFetch(expired).IsError)
		var listTotal, getTotal time.Duration
		compiledBefore := host.windowPrimitiveSource.metadataCompiles.Load()
		materializedBefore := host.windowPrimitiveSource.snapshot.CompileCount()
		var bindingTotal, sourceTotal time.Duration
		for i := 0; i < 7; i++ {
			observationCtx := oauth.WithBearer(ctx, token.Load().(string))
			start := time.Now()
			_, err = host.ResolveComponentBinding(observationCtx, ref)
			require.NoError(t, err)
			bindingTotal += time.Since(start)
			parsed, err := identity.ParseResourceURI(uri)
			require.NoError(t, err)
			start = time.Now()
			_, err = host.windowPrimitiveSource.bytes(observationCtx, parsed)
			require.NoError(t, err)
			sourceTotal += time.Since(start)
			start = time.Now()
			_, err = gateway.List(ctx, "window", "team")
			require.NoError(t, err)
			listTotal += time.Since(start)
			start = time.Now()
			_, err = gateway.Get(ctx, listed[0].Connection, identity.ResourceRef{URI: uri}, got.ResolvedResource)
			require.NoError(t, err)
			getTotal += time.Since(start)
		}
		t.Logf("fixture cached discovery: Binding mean=%s Source mean=%s Gateway.List mean=%s Gateway.Get mean=%s canonicalBytes=%d", bindingTotal/7, sourceTotal/7, listTotal/7, getTotal/7, len(got.Resource.DefinitionBytes))
		require.Equal(t, compiledBefore, host.windowPrimitiveSource.metadataCompiles.Load())
		require.Equal(t, materializedBefore, host.windowPrimitiveSource.snapshot.CompileCount())
		require.EqualValues(t, 1, compiledBefore, "shared component metadata compiles once at startup")
		token.Store(f.token(t, f.private, "alice", []string{"visitor"}, time.Now().Add(time.Minute)))
		denied, err := gateway.List(ctx, "window", "team")
		require.NoError(t, err)
		require.Empty(t, denied)
		_, err = gateway.Get(ctx, listed[0].Connection, identity.ResourceRef{URI: uri}, got.ResolvedResource)
		require.Error(t, err)
		require.True(t, *callFetch(*got.ResolvedResource).IsError)
		token.Store(f.token(t, f.private, "alice", []string{"reader"}, time.Now().Add(time.Minute)))
		// Current native metadata rows are observed on every operation; the read
		// plan cache cannot hide changed source DQL or silently repin a reader.
		var oldDQL string
		require.NoError(t, host.studio.QueryRow("SELECT generated_dql FROM component_versions WHERE report_id='records' AND version_no=1").Scan(&oldDQL))
		_, err = host.studio.Exec("UPDATE component_versions SET generated_dql=? WHERE report_id='records' AND version_no=1", oldDQL+"\n-- changed stored source")
		require.NoError(t, err)
		_, err = gateway.Get(ctx, listed[0].Connection, identity.ResourceRef{URI: uri}, got.ResolvedResource)
		require.Error(t, err)
		_, err = host.studio.Exec("UPDATE component_versions SET generated_dql=? WHERE report_id='records' AND version_no=1", oldDQL)
		require.NoError(t, err)
		_, err = gateway.Get(ctx, listed[0].Connection, identity.ResourceRef{URI: uri}, nil)
		require.Error(t, err, "observed source drift remains stale until restart, even if old rows are restored")
		require.NoError(t, os.WriteFile(sourceFile, []byte(strings.Replace(string(window), "Studio configured window", "Changed configured window", 1)), 0600))
		_, err = gateway.Get(ctx, listed[0].Connection, identity.ResourceRef{URI: uri}, got.ResolvedResource)
		require.Error(t, err, "changed configured presentation cannot upgrade an open pin")
		require.True(t, *callFetch(*got.ResolvedResource).IsError)
		refPin := *got.ResolvedResource
		refPin.ContentFingerprint = "forged"
		_, err = gateway.Get(ctx, listed[0].Connection, identity.ResourceRef{URI: uri}, &refPin)
		require.Error(t, err)
		require.True(t, *callFetch(refPin).IsError)
	}})
}

func configWindowPolicy(policy authz.Policy) authz.Document {
	return authz.Document{Resource: authz.Resource{Kind: "window", ID: "team/overview", Version: identity.WorkingCandidate, Tenant: "*"}, Revision: 1, Policies: map[string]authz.Policy{"discover": policy, "describe": policy, "execute": policy}}
}
