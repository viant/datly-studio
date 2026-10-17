package host

import (
	"context"
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
	"github.com/viant/authz/gating"
	"github.com/viant/authz/oauth"
	"github.com/viant/forge/backend/clientpool"
	bridge "github.com/viant/forge/backend/provider"
	primitive "github.com/viant/forge/backend/provider/contract"
	identity "github.com/viant/forge/backend/resource"
	coreresource "github.com/viant/forge/backend/resourcehost"
	"github.com/viant/forge/backend/types"
	"github.com/viant/jsonrpc/transport/client/http/streamable"
	"github.com/viant/mcp-protocol/schema"
	mcpclient "github.com/viant/mcp/client"
)

type shortenedDatasourceLease struct {
	*oauth.Provider
	until         atomic.Int64
	awaitDeadline atomic.Bool
	canceled      atomic.Bool
}

func (p *shortenedDatasourceLease) Resolve(ctx context.Context) (authz.Facts, error) {
	principal, err := p.ResolvePrincipal(ctx)
	return principal.Facts, err
}

func (p *shortenedDatasourceLease) ResolvePrincipal(ctx context.Context) (gating.Principal, error) {
	principal, err := p.Provider.ResolvePrincipal(ctx)
	if err != nil {
		return principal, err
	}
	if value := p.until.Load(); value != 0 {
		until := time.Unix(0, value)
		if principal.Facts.ValidUntil.After(until) {
			principal.Facts.ValidUntil = until
		}
		if deadline, ok := ctx.Deadline(); ok && !deadline.After(until) && p.awaitDeadline.CompareAndSwap(true, false) {
			<-ctx.Done()
			p.canceled.Store(true)
			return gating.Principal{}, ctx.Err()
		}
	}
	return principal, nil
}

func TestConfiguredDatasourcePublicationMCPProofAndNativeSQL(t *testing.T) {
	f := newStockOAuthFixture(t)
	leaseProvider := &shortenedDatasourceLease{Provider: f.provider}
	var token atomic.Value
	token.Store(f.token(t, f.private, "alice", []string{"reader"}, time.Now().Add(time.Minute)))
	uri := "window://team/overview"
	datasourceURI := "datasource://team/overview/records"
	ref := ComponentReference{Kind: "dynamic", ID: "records", Revision: "1", Method: http.MethodGet, Route: "/records"}
	// Existing Forge Window/Table models are the renderer contract, unchanged.
	window := []byte("window:\n  view:\n    content:\n      id: original-records\n      title: Studio records\n      dataSourceRef: records\n      fetchData: true\n      table:\n        columns:\n          - id: Id\n            name: ID\n          - id: Name\n            name: Name\ndataSources:\n  records:\n    id: records\n    selectors:\n      data: Records\n    cardinality: many\n    parameters: []\n")
	rule := authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}
	testDynamicHost(t, true, false, dynamicHostExtension{configure: func(config *Config) {
		config.Access = &ResourceAccessConfig{Tenant: "*", Provider: leaseProvider}
		config.Forge = &ForgeConfig{ProviderIdentity: "studio-fixture", Windows: []ForgeWindow{{URI: uri, DefinitionPath: "records-window.yaml", Components: map[string]ComponentReference{"records": ref}, PublishDataSources: true}}, Policies: []authz.Document{{Resource: authz.Resource{Kind: "window", ID: "team/overview", Version: identity.WorkingCandidate, Tenant: "*"}, Revision: 1, Policies: map[string]authz.Policy{"discover": rule, "describe": rule, "execute": rule}}}, Selections: []authz.SelectionDocument{{Resource: authz.ResourceFamily{Kind: "window", ID: "team/overview", Tenant: "*"}, Revision: 1, DefaultVersion: identity.WorkingCandidate}}}
		require.NoError(t, os.WriteFile(filepath.Join(config.RootDir, "records-window.yaml"), window, 0600))
	}, verify: func(host *Service, address string) {
		store, err := authz.NewStaticStore([]authz.Document{{Resource: authz.Resource{Kind: "component", ID: "records", Version: "1", Tenant: "*"}, Revision: 1, Policies: map[string]authz.Policy{"describe": rule, "execute": rule}}})
		require.NoError(t, err)
		host.resourceAccess.Store = store
		ctx := context.Background()
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
			if request.Method == schema.MethodResourcesRead {
				headers.Set(schema.HeaderName, request.Params.URI)
			}
			return nil
		}))
		require.NoError(t, err)
		client := mcpclient.New("datasource-publication", "1", transport, mcpclient.WithProtocolVersion(schema.LatestProtocolVersion))
		defer client.Close()
		_, err = client.Initialize(ctx)
		require.NoError(t, err)
		call := func(name string, args map[string]any) json.RawMessage {
			t.Helper()
			result, err := client.CallTool(ctx, &schema.CallToolRequestParams{Name: name, Arguments: args}, mcpclient.WithNoRetry())
			require.NoError(t, err)
			require.False(t, result.IsError != nil && *result.IsError, "%s: %+v", name, result)
			raw, err := json.Marshal(result.StructuredContent)
			require.NoError(t, err)
			return raw
		}
		tools, err := client.ListTools(ctx, nil)
		require.NoError(t, err)
		seen := map[string]bool{}
		for _, tool := range tools.Tools {
			seen[tool.Name] = true
		}
		for _, name := range []string{"datasources/list", "datasources/get", "datasources/fetch"} {
			require.True(t, seen[name])
		}
		capabilities := call("namespaces/get", map[string]any{"namespace": "team"})
		require.Contains(t, string(capabilities), "datasources/fetch")
		_, directErr := host.windowPrimitives.List(oauth.WithBearer(ctx, token.Load().(string)), "datasource", primitive.ListRequest{Namespace: "team"})
		require.NoError(t, directErr)
		listed := call("datasources/list", map[string]any{"namespace": "team"})
		require.Contains(t, string(listed), datasourceURI)
		windowBytes := call("windows/get", map[string]any{"uri": uri, "revision": "working"})
		var gotWindow primitive.GetResult
		require.NoError(t, json.Unmarshal(windowBytes, &gotWindow))
		variant, err := types.SelectWindowResourceWithReferences(gotWindow.Resource.DefinitionBytes, nil)
		require.NoError(t, err)
		require.Equal(t, "original-records", variant.Window.View.Content.ID)
		require.Equal(t, datasourceURI, variant.DataSourceResources["records"].Resource.URI)
		require.NotNil(t, variant.Window.View.Content.Table)
		require.Len(t, variant.Window.View.Content.Table.Columns, 2)
		require.NotNil(t, variant.Window.View.Content.FetchData)
		require.True(t, *variant.Window.View.Content.FetchData)
		datasourceBytes := call("datasources/get", map[string]any{"uri": datasourceURI, "revision": "working"})
		var got primitive.GetResult
		require.NoError(t, json.Unmarshal(datasourceBytes, &got))
		require.NotNil(t, got.ExecutionProof)
		require.Equal(t, got.ResolvedResource.ContentFingerprint, variant.DataSourceResources["records"].ContentFingerprint)
		var definition stockDatasourceDefinition
		require.NoError(t, json.Unmarshal(got.Resource.DefinitionBytes, &definition))
		require.Equal(t, "", definition.DataSource.Backend.Service)
		require.Equal(t, "datasources/fetch", definition.DataSource.Backend.Method)
		require.Equal(t, "1", definition.DataSource.Backend.Component.Revision)
		resources, err := client.ListResources(ctx, nil)
		require.NoError(t, err)
		require.Len(t, resources.Resources, 1)
		require.Equal(t, datasourceURI, resources.Resources[0].Uri)
		read, err := client.ReadResource(ctx, &schema.ReadResourceRequestParams{Uri: datasourceURI})
		require.NoError(t, err)
		require.Len(t, read.Contents, 1)
		require.Equal(t, string(got.Resource.DefinitionBytes), read.Contents[0].Text)
		input := stockPublishedDatasourceInput{Resource: *got.ResolvedResource, ExecutionProof: got.ExecutionProof, DataSourceID: "records", Inputs: map[string]any{}}
		encoded, _ := json.Marshal(input)
		var args map[string]any
		require.NoError(t, json.Unmarshal(encoded, &args))
		data := call("datasources/fetch", args)
		require.Contains(t, string(data), "ready")
		require.Contains(t, string(data), "primary")
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
			if err != nil || original.Subject != current.Subject || original.IdentityRevision != current.IdentityRevision || !original.Valid(time.Now()) {
				return identity.ErrResourceDenied
			}
			return nil
		}, "gateway-not-provider")
		defer gateway.Close()
		resolvedDescriptor, resolvedProof, err := gateway.ResolveDatasource(ctx, *got.ResolvedResource)
		require.NoError(t, err)
		fetched, err := gateway.FetchDatasource(ctx, *got.ResolvedResource, resolvedProof, resolvedDescriptor, map[string]interface{}{})
		require.NoError(t, err)
		require.Contains(t, string(fetched), "ready")
		require.Contains(t, string(fetched), "primary")
		hostProof, err := types.NewWindowTargetHMAC([]byte(strings.Repeat("p", 32)))
		require.NoError(t, err)
		catalog := &coreresource.WindowCatalog{Gateway: gateway, TargetProof: hostProof}
		opened, err := catalog.Get(ctx, &bridge.WindowDefinitionGetInput{WindowID: uri})
		require.NoError(t, err)
		require.Len(t, opened.Definition.ResourceTarget.DependencyPins, 1)
		require.Equal(t, datasourceURI, opened.Definition.ResourceTarget.DependencyPins["records"].URI)
		parentDescriptor, err := catalog.ResolveDatasource(ctx, *opened.Definition.Resource, opened.Definition.ResourceTarget, "records")
		require.NoError(t, err)
		parentData, err := catalog.FetchProviderDatasource(ctx, *opened.Definition.Resource, opened.Definition.ResourceTarget, parentDescriptor, map[string]interface{}{})
		require.NoError(t, err)
		require.Contains(t, string(parentData), "ready")
		require.Contains(t, string(parentData), "primary")
		for _, mutation := range []string{"alias", "uri", "actor", "lease", "proof"} {
			changed := input
			proof := *input.ExecutionProof
			changed.ExecutionProof = &proof
			switch mutation {
			case "alias":
				changed.DataSourceID = "caller-alias"
			case "uri":
				changed.Resource.URI = "datasource://team/unknown/records"
			case "actor":
				changed.Resource.AuthorityBinding = "foreign"
			case "lease":
				changed.Resource.ValidUntil = proof.Resource.ValidUntil.Add(time.Second)
			case "proof":
				proof.Binding = "window-proof-cannot-authorize-datasource"
			}
			encoded, _ := json.Marshal(changed)
			var altered map[string]any
			require.NoError(t, json.Unmarshal(encoded, &altered))
			result, err := client.CallTool(ctx, &schema.CallToolRequestParams{Name: "datasources/fetch", Arguments: altered}, mcpclient.WithNoRetry())
			require.NoError(t, err)
			require.True(t, result.IsError != nil && *result.IsError, mutation)
		}
		leaseProvider.until.Store(time.Now().Add(5 * time.Second).UnixNano())
		leaseProvider.awaitDeadline.Store(true)
		result, err := client.CallTool(ctx, &schema.CallToolRequestParams{Name: "datasources/fetch", Arguments: args}, mcpclient.WithNoRetry())
		require.NoError(t, err)
		require.True(t, result.IsError != nil && *result.IsError)
		require.True(t, leaseProvider.canceled.Load(), "fresh shorter lease must bound native execution context")
		encodedResult, err := json.Marshal(result)
		require.NoError(t, err)
		require.NotContains(t, string(encodedResult), "ready")
		leaseProvider.until.Store(0)
		token.Store(f.token(t, f.private, "mallory", []string{"outsider"}, time.Now().Add(time.Minute)))
		_, err = client.ReadResource(ctx, &schema.ReadResourceRequestParams{Uri: datasourceURI})
		require.Error(t, err)
	}})
}
