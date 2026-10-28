package catalog

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/runtime/host"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	stored "github.com/viant/datly-studio/studio/namespaces/store_read"
	"github.com/viant/datly/exec"
	"github.com/viant/forge/backend/artifactcohort"
	primitive "github.com/viant/forge/backend/provider/contract"
	"github.com/viant/jsonrpc"
	"github.com/viant/jsonrpc/transport"
	"github.com/viant/jsonrpc/transport/client/http/streamable"
	protoclient "github.com/viant/mcp-protocol/client"
	namespaceext "github.com/viant/mcp-protocol/extension/namespaces"
	"github.com/viant/mcp-protocol/logger"
	"github.com/viant/mcp-protocol/schema"
	protoserver "github.com/viant/mcp-protocol/server"
	mcpclient "github.com/viant/mcp/client"
	mcpserver "github.com/viant/mcp/server"
	"github.com/viant/scy/auth/jwt"
	xresponse "github.com/viant/xdatly/response"
)

type nativeSyncPrincipalKey struct{}
type nativeSyncDirectory struct{ row *stored.StoredNamespace }

func (d nativeSyncDirectory) InvokeComponent(_ context.Context, request exec.ComponentRequest) (any, error) {
	input, ok := request.Input.(*stored.Input)
	if !ok || !input.Scoped || input.Subject == "" {
		return nil, errors.New("verified scoped namespace read required")
	}
	rows := []*stored.StoredNamespace{}
	if input.NamespaceId == "" || input.NamespaceId == d.row.NamespaceId {
		rows = append(rows, d.row)
		if input.NamespaceId == "" {
			rows = append(rows, &stored.StoredNamespace{NamespaceId: strings.Repeat("b", 64), Name: "unrelated", Status: "active"})
		}
	}
	return &stored.Output{Namespaces: rows}, nil
}

type countedNativeSyncSource struct {
	artifactcohort.Caller
	discoveries atomic.Int64
}

func (s *countedNativeSyncSource) ArtifactNamespaces() []string {
	if source, ok := s.Caller.(interface{ ArtifactNamespaces() []string }); ok {
		return source.ArtifactNamespaces()
	}
	return nil
}
func (s *countedNativeSyncSource) Discover(ctx context.Context) (artifactcohort.Manifest, error) {
	s.discoveries.Add(1)
	return s.Caller.Discover(ctx)
}

func TestNativeNamespaceSynchronizationOverMCPKeepsOpaqueURIAndHostBoundary(t *testing.T) {
	retainManagementSettings(t)
	t.Setenv("STUDIO_MANAGEMENT_NAMESPACE", "")
	defer ConfigurePublishedArtifactCohorts(nil, nil)
	ctx := context.Background()
	catalog := host.NewPublishedArtifactCatalog(artifactcohort.NewPublisher())
	root := artifactcohort.RootRef{URI: "report://example/summary", Kind: "report", Version: 1}
	artifact, generation, err := catalog.Publish(ctx, root, json.RawMessage(`{"schemaVersion":1,"reportSpec":{}}`), func(context.Context, json.RawMessage) error { return nil })
	require.NoError(t, err)
	prompt, gen, err := catalog.Publish(ctx, artifactcohort.RootRef{URI: "prompt://example/help", Kind: "prompt", Version: 1}, json.RawMessage(`{"schemaVersion":1,"text":"Example"}`), func(context.Context, json.RawMessage) error { return nil })
	require.NoError(t, err)
	generation = gen
	policy := authz.Resource{Tenant: "tenant", Kind: "cohort", ID: "readers", Version: "logical"}
	manifest, err := catalog.ReplaceCatalog(ctx, []authz.Document{{Resource: policy, Revision: 1, Policies: map[string]authz.Policy{artifactcohort.PolicyAction: {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: "*"}}}}}, []artifactcohort.Cohort{{Tenant: "tenant", Name: "readers", PolicyResource: policy, Roots: []artifactcohort.RootRef{artifact.Root, prompt.Root}}}, generation)
	require.NoError(t, err)
	counted := &countedNativeSyncSource{Caller: catalog}
	require.NoError(t, ConfigurePublishedArtifactCohorts(counted, func(ctx context.Context) error {
		if ctx.Value(nativeSyncPrincipalKey{}) != "host" {
			return errors.New("configured host required")
		}
		return nil
	}))
	opaqueID := strings.Repeat("a", 64)
	session := releaseTestSession{binder: releaseTestBinder{invoker: nativeSyncDirectory{row: &stored.StoredNamespace{NamespaceId: opaqueID, Name: "example", Status: "active"}}}}
	signer := datatest.NewJWTFixture(t)
	block, _ := pem.Decode(signer.PublicKeyPEM(t))
	public, err := x509.ParsePKIXPublicKey(block.Bytes)
	require.NoError(t, err)
	protocol, err := mcpserver.New(mcpserver.WithNewHandler(func(_ context.Context, notifier transport.Notifier, log logger.Logger, client protoclient.Operations) (protoserver.Handler, error) {
		handler := protoserver.NewDefaultHandler(notifier, log, client)
		register := func(name string, run func(context.Context, map[string]any) (any, error)) {
			handler.Registry.RegisterToolWithSchema(name, "Synthetic namespace contract", schema.ToolInputSchema{Type: "object"}, nil, func(ctx context.Context, request *schema.CallToolRequest) (*schema.CallToolResult, *jsonrpc.Error) {
				value, err := run(ctx, request.Params.Arguments)
				failed := err != nil
				if failed {
					if body, ok := xresponse.ErrorBody(err); ok {
						value = body
					} else {
						value = map[string]string{"message": "namespace unavailable"}
					}
				}
				raw, _ := json.Marshal(value)
				return &schema.CallToolResult{IsError: &failed, Content: []schema.CallToolResultContentElem{schema.TextContent{Type: "text", Text: string(raw)}}}, nil
			})
		}
		verified := func(ctx context.Context) (*jwt.Claims, *studioauth.Output) {
			subject, _ := ctx.Value(nativeSyncPrincipalKey{}).(string)
			claims := &jwt.Claims{}
			claims.Subject = subject
			return claims, &studioauth.Output{Auth: &studioauth.AuthContext{Subject: subject}}
		}
		register("namespaces/list", func(ctx context.Context, args map[string]any) (any, error) {
			claims, auth := verified(ctx)
			raw, _ := json.Marshal(args)
			input := &ListInput{Jwt: claims, Auth: auth}
			_ = json.Unmarshal(raw, &input.Request)
			out := &ListOutput{}
			return out, NewList().Exec(ctx, session, input, out)
		})
		register("namespaces/get", func(ctx context.Context, args map[string]any) (any, error) {
			claims, auth := verified(ctx)
			raw, _ := json.Marshal(args)
			var request struct {
				namespaceext.GetRequest
				Namespace string                          `json:"namespace"`
				Sync      *primitive.NamespaceSyncRequest `json:"sync"`
			}
			if err := json.Unmarshal(raw, &request); err != nil {
				return nil, err
			}
			out := &GetOutput{}
			return out, NewGet().Exec(ctx, session, &GetInput{Jwt: claims, Auth: auth, Request: NamespaceGetRequest{URI: request.URI, Kind: request.Kind, Cursor: request.Cursor}, Namespace: request.Namespace, Sync: request.Sync}, out)
		})
		register(artifactcohort.MethodFetch, func(ctx context.Context, args map[string]any) (any, error) {
			claims, auth := verified(ctx)
			raw, _ := json.Marshal(args)
			input := &ArtifactFetchInput{Jwt: claims, Auth: auth}
			if err := json.Unmarshal(raw, input); err != nil {
				return nil, err
			}
			out := &ArtifactFetchOutput{}
			return out, NewArtifactFetch().Exec(ctx, session, input, out)
		})
		return handler, nil
	}))
	require.NoError(t, err)
	protocol.UseStreamableHTTP(true)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parsed, err := jwtv5.Parse(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), func(*jwtv5.Token) (any, error) { return public, nil }, jwtv5.WithValidMethods([]string{"RS256"}))
		if err != nil || !parsed.Valid {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		subject, err := parsed.Claims.GetSubject()
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		protocol.HTTP(ctx, "").Handler.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), nativeSyncPrincipalKey{}, subject)))
	}))
	defer server.Close()
	clientFor := func(subject string) mcpclient.Interface {
		bearer := signer.Bearer(t, subject)
		wire, err := streamable.New(ctx, server.URL+"/mcp", streamable.WithStateless(), streamable.WithProtocolVersion(schema.LatestProtocolVersion), streamable.WithRequestHeaderProvider(func(_ context.Context, body []byte, headers http.Header) error {
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
			headers.Set("Authorization", bearer)
			return nil
		}))
		require.NoError(t, err)
		client := mcpclient.New("native-namespace-test", "1", wire, mcpclient.WithProtocolVersion(schema.LatestProtocolVersion))
		_, err = client.Initialize(ctx)
		require.NoError(t, err)
		t.Cleanup(func() { client.Close() })
		return client
	}
	hostClient, viewer := clientFor("host"), clientFor("viewer")
	call := func(client mcpclient.Interface, name string, args map[string]any) json.RawMessage {
		result, err := client.CallTool(ctx, &schema.CallToolRequestParams{Name: name, Arguments: args}, mcpclient.WithNoRetry())
		require.NoError(t, err)
		raw, _ := json.Marshal(result.Content[0])
		var text struct {
			Text string `json:"text"`
		}
		require.NoError(t, json.Unmarshal(raw, &text))
		return json.RawMessage(text.Text)
	}
	listed := call(viewer, "namespaces/list", map[string]any{})
	var index namespaceListWire
	require.NoError(t, json.Unmarshal(listed, &index))
	require.Len(t, index.Namespaces, 2)
	require.Equal(t, "ns://"+opaqueID, index.Namespaces[0].URI)
	require.Equal(t, "example", index.Namespaces[0].Name)
	require.NotNil(t, index.Namespaces[0].Synchronization)
	require.Nil(t, index.Namespaces[1].Synchronization)
	require.NotContains(t, string(listed), "policies")
	require.Zero(t, counted.discoveries.Load())
	ordinary := call(viewer, "namespaces/get", map[string]any{"uri": "ns://" + opaqueID})
	require.NotContains(t, string(ordinary), "sync")
	require.NotContains(t, string(ordinary), "policies")
	require.Zero(t, counted.discoveries.Load())
	caller := artifactcohort.NamespaceCaller{Client: hostClient, Namespace: "example"}
	full, err := caller.Discover(ctx)
	require.NoError(t, err)
	require.Equal(t, manifest.Hash, full.Hash)
	require.Equal(t, manifest.PartitionHashes, full.PartitionHashes)
	require.NotEmpty(t, full.PartitionHashes["example|report"])
	require.NotEmpty(t, full.PartitionHashes["example|prompt"])
	require.NotEmpty(t, full.PartitionHashes["example|policy"])
	require.Len(t, full.Cohorts, 1)
	compact, err := caller.DiscoverKnown(ctx, manifest.Hash)
	require.NoError(t, err)
	require.True(t, compact.Unchanged)
	require.Empty(t, compact.Cohorts)
	compactWire := call(hostClient, "namespaces/get", map[string]any{"namespace": "example", "sync": map[string]any{"knownHash": manifest.Hash}})
	var envelope namespaceGetWire
	require.NoError(t, json.Unmarshal(compactWire, &envelope))
	var compactFields map[string]any
	require.NoError(t, json.Unmarshal(envelope.Sync.Catalog, &compactFields))
	require.Len(t, compactFields, 2)
	byURI := call(hostClient, "namespaces/get", map[string]any{"uri": "ns://" + opaqueID, "sync": map[string]any{}})
	var synced namespaceGetWire
	require.NoError(t, json.Unmarshal(byURI, &synced))
	var same artifactcohort.Manifest
	require.NoError(t, json.Unmarshal(synced.Sync.Catalog, &same))
	require.Equal(t, full.Hash, same.Hash)
	fetched, err := caller.Fetch(ctx, artifactcohort.FetchInput{Tenant: "tenant", Root: artifact.Root})
	require.NoError(t, err)
	require.Equal(t, artifact.Root, fetched.Root)
	denied, err := (artifactcohort.NamespaceCaller{Client: viewer, Namespace: "example"}).Discover(ctx)
	require.Error(t, err)
	require.Empty(t, denied.Hash)
	malformed := call(hostClient, "namespaces/get", map[string]any{"namespace": "example", "sync": map[string]any{"knownHash": "bad"}})
	require.Contains(t, string(malformed), "namespace_sync_request_invalid")
	require.NotEqual(t, "null", string(malformed))
	foreign := call(hostClient, "namespaces/get", map[string]any{"namespace": "other", "sync": map[string]any{}})
	require.NotContains(t, string(foreign), manifest.Hash)
	require.NoError(t, ConfigurePublishedArtifactCohorts(nil, nil))
	disabled := call(hostClient, "namespaces/get", map[string]any{"namespace": "example", "sync": map[string]any{}})
	require.Contains(t, string(disabled), "namespace_sync_unavailable")
	require.Contains(t, string(disabled), "not configured")
	require.NotEqual(t, "null", string(disabled))
}
