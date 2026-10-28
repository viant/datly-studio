package host

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/viant/authz"
	"github.com/viant/forge/backend/artifactcohort"
	primitive "github.com/viant/forge/backend/provider/contract"
	identity "github.com/viant/forge/backend/resource"
	"github.com/viant/forge/backend/types"
	"github.com/viant/jsonrpc/transport/client/http/streamable"
	"github.com/viant/mcp-protocol/schema"
	mcpclient "github.com/viant/mcp/client"
)

func TestConfiguredNamespaceSyncAdvertisementAndConditionalCatalogOverMCP(t *testing.T) {
	ctx := context.Background()
	oauth := newStockOAuthFixture(t)
	publisher := NewPublishedArtifactCatalog(artifactcohort.NewPublisher())
	root := artifactcohort.RootRef{URI: "window://example/published", Kind: "window", Version: 1}
	artifact, generation, err := publisher.Publish(ctx, root, closedWindowForCohort(t, "published-root"), func(_ context.Context, raw json.RawMessage) error {
		var envelope types.WindowResourceEnvelope
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return err
		}
		return envelope.Validate()
	})
	require.NoError(t, err)
	policyRef := authz.Resource{Tenant: "tenant", Kind: "cohort", ID: "readers", Version: "logical"}
	manifest, err := publisher.ReplaceCatalog(ctx, []authz.Document{{Resource: policyRef, Revision: 1, Policies: map[string]authz.Policy{artifactcohort.PolicyAction: {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: "*"}}}}}, []artifactcohort.Cohort{{Tenant: "tenant", Name: "readers", PolicyResource: policyRef, Roots: []artifactcohort.RootRef{artifact.Root}}}, generation)
	require.NoError(t, err)
	var token atomic.Value
	token.Store(oauth.token(t, oauth.private, "host", []string{"reader"}, time.Now().Add(time.Minute)))
	readerPolicy := authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}
	testDynamicHost(t, true, false, dynamicHostExtension{skipDefaultChecks: true, configure: func(config *Config) {
		config.Access = &ResourceAccessConfig{Tenant: "*", Provider: oauth.provider}
		config.ArtifactCohorts = publisher
		config.AuthorizeArtifactHost = func(ctx context.Context) error {
			principal, err := oauth.provider.ResolvePrincipal(ctx)
			if err != nil || principal.Facts.Subject != "host" {
				return errors.New("configured host required")
			}
			return nil
		}
		config.Forge = &ForgeConfig{ProviderIdentity: "synthetic-producer", Windows: []ForgeWindow{{URI: "window://example/overview", DefinitionPath: "namespace-window.yaml", Components: map[string]ComponentReference{"records": {Kind: "dynamic", ID: "records", Revision: "1", Method: http.MethodGet, Route: "/records"}}}}, Policies: []authz.Document{{Resource: authz.Resource{Tenant: "*", Kind: "window", ID: "example/overview", Version: identity.WorkingCandidate}, Revision: 1, Policies: map[string]authz.Policy{"discover": readerPolicy, "describe": readerPolicy, "execute": readerPolicy}}}, Selections: []authz.SelectionDocument{{Resource: authz.ResourceFamily{Tenant: "*", Kind: "window", ID: "example/overview"}, Revision: 1, DefaultVersion: identity.WorkingCandidate}}}
		config.Forge.Windows = append(config.Forge.Windows, ForgeWindow{URI: "window://unrelated/overview", DefinitionPath: "namespace-window.yaml", Components: map[string]ComponentReference{"records": {Kind: "dynamic", ID: "records", Revision: "1", Method: http.MethodGet, Route: "/records"}}})
		config.Forge.Policies = append(config.Forge.Policies, authz.Document{Resource: authz.Resource{Tenant: "*", Kind: "window", ID: "unrelated/overview", Version: identity.WorkingCandidate}, Revision: 1, Policies: map[string]authz.Policy{"discover": readerPolicy, "describe": readerPolicy, "execute": readerPolicy}})
		config.Forge.Selections = append(config.Forge.Selections, authz.SelectionDocument{Resource: authz.ResourceFamily{Tenant: "*", Kind: "window", ID: "unrelated/overview"}, Revision: 1, DefaultVersion: identity.WorkingCandidate})
		require.NoError(t, os.WriteFile(filepath.Join(config.RootDir, "namespace-window.yaml"), []byte("window:\n  view:\n    content:\n      id: namespace-root\ndataSources:\n  records:\n    id: records\n    selectors:\n      data: Records\n    cardinality: many\n    parameters: []\n"), 0600))
	}, verify: func(_ *Service, address string) {
		transport, err := streamable.New(ctx, "http://"+address+"/forge/mcp", streamable.WithStateless(), streamable.WithProtocolVersion(schema.LatestProtocolVersion), streamable.WithHTTPClient(&http.Client{Transport: studioBearerTransport{token: &token}}), streamable.WithRequestHeaderProvider(func(_ context.Context, raw []byte, headers http.Header) error {
			var request struct {
				Method string
				Params struct{ Name string }
			}
			if err := json.Unmarshal(raw, &request); err != nil {
				return err
			}
			headers.Set(schema.HeaderMethod, request.Method)
			if request.Method == schema.MethodToolsCall {
				headers.Set(schema.HeaderName, request.Params.Name)
			}
			return nil
		}))
		require.NoError(t, err)
		client := mcpclient.New("namespace-sync-wire", "1", transport, mcpclient.WithProtocolVersion(schema.LatestProtocolVersion))
		defer client.Close()
		_, err = client.Initialize(ctx)
		require.NoError(t, err)
		listed, err := client.CallTool(ctx, &schema.CallToolRequestParams{Name: "namespaces/list", Arguments: map[string]any{}}, mcpclient.WithNoRetry())
		require.NoError(t, err)
		raw, err := json.Marshal(listed.StructuredContent)
		require.NoError(t, err)
		var namespaces primitive.NamespaceListResult
		require.NoError(t, json.Unmarshal(raw, &namespaces))
		require.Len(t, namespaces.Namespaces, 2)
		require.Equal(t, "example", namespaces.Namespaces[0].Name)
		require.NotNil(t, namespaces.Namespaces[0].Synchronization)
		require.Nil(t, namespaces.Namespaces[1].Synchronization)
		require.Equal(t, "namespaces/get", namespaces.Namespaces[0].Synchronization.Method)
		require.NotContains(t, string(raw), "policies")
		syncWire, err := client.CallTool(ctx, &schema.CallToolRequestParams{Name: "namespaces/get", Arguments: map[string]any{"namespace": "example", "sync": map[string]any{}}}, mcpclient.WithNoRetry())
		require.NoError(t, err)
		syncRaw, err := json.Marshal(syncWire.StructuredContent)
		require.NoError(t, err)
		var advertised primitive.NamespaceCapabilities
		require.NoError(t, json.Unmarshal(syncRaw, &advertised))
		require.Equal(t, "synthetic-producer", advertised.ProviderIdentity)
		caller := artifactcohort.NamespaceCaller{Client: client, Namespace: "example"}
		full, err := caller.Discover(ctx)
		require.NoError(t, err)
		require.Equal(t, manifest.Hash, full.Hash)
		require.Equal(t, manifest.PartitionHashes, full.PartitionHashes)
		compact, err := caller.DiscoverKnown(ctx, manifest.Hash)
		require.NoError(t, err)
		require.True(t, compact.Unchanged)
		require.Empty(t, compact.Cohorts)
		token.Store(oauth.token(t, oauth.private, "viewer", []string{"reader"}, time.Now().Add(time.Minute)))
		_, err = caller.Discover(ctx)
		require.Error(t, err)
		denied, err := client.CallTool(ctx, &schema.CallToolRequestParams{Name: "namespaces/get", Arguments: map[string]any{"namespace": "example", "sync": map[string]any{}}}, mcpclient.WithNoRetry())
		require.NoError(t, err)
		require.NotNil(t, denied.IsError)
		require.True(t, *denied.IsError)
		diagnostic, _ := json.Marshal(denied.StructuredContent)
		require.Contains(t, string(diagnostic), "namespace_sync_denied")
		require.Contains(t, string(diagnostic), "host access")
	}})
}
