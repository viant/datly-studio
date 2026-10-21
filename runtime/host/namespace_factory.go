package host

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/viant/datly-studio/audit"
	"io/fs"
	"maps"
	"net/http"
	"net/url"
	"strings"

	"github.com/viant/datly-studio/runtime/namespacemcp"
	mcpserver "github.com/viant/datly/mcp/server"
	"github.com/viant/mcp-protocol/schema"
)

// NamespaceFactory builds an independent runtime manager for each namespace.
// The returned source owns its runtime resources and closes after listener drain.
func NamespaceFactory(base Config) namespacemcp.Factory {
	return func(ctx context.Context, endpoint namespacemcp.Endpoint) (mcpserver.Config, error) {
		config := base
		config.NamespaceID = endpoint.NamespaceID
		config.MCP.Address = endpoint.Address
		config.Authentication.PublicMCPURL = "http://" + endpoint.Address
		config.NamespaceMCPPath = strings.TrimSuffix(endpoint.Path, "/mcp")
		config.Authentication.Providers = maps.Clone(base.Authentication.Providers)
		config.Authentication.Components = maps.Clone(base.Authentication.Components)
		runtime, err := New(ctx, config)
		if err != nil {
			return mcpserver.Config{}, err
		}
		if err = runtime.Reload(ctx, 0); err != nil {
			_ = runtime.Close(context.Background())
			return mcpserver.Config{}, err
		}
		return mcpserver.Config{Source: &namespaceRuntimeSource{runtime: runtime, endpoint: endpoint}, Implementation: schema.Implementation{Name: "datly-studio-namespace", Version: "1"}, Transport: mcpserver.TransportConfig{Kind: mcpserver.TransportStreamable, Address: endpoint.Address, CORS: config.MCP.CORS}}, nil
	}
}

type namespaceRuntimeSource struct {
	runtime  *Service
	endpoint namespacemcp.Endpoint
}

func (s *namespaceRuntimeSource) Pin(ctx context.Context) (context.Context, mcpserver.ServerService, error) {
	return s.runtime.manager.Pin(ctx)
}
func (s *namespaceRuntimeSource) PinSnapshot(ctx context.Context) (context.Context, mcpserver.ServerService, error) {
	return s.runtime.manager.PinSnapshot(ctx)
}
func (s *namespaceRuntimeSource) WrapHTTP(next http.Handler) http.Handler {
	if wrap := s.runtime.config.NamespaceHTTPMiddleware; wrap != nil {
		next = wrap(s.endpoint, next)
	}
	return s.WrapNativeHTTP(next)
}

// WrapNativeHTTP protects the native transport independently of host composition.
func (s *namespaceRuntimeSource) WrapNativeHTTP(next http.Handler) http.Handler {
	if s.runtime.config.ToolCallAudit != nil {
		wrapped := s.runtime.config.ToolCallAudit.Wrap(next)
		next = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrapped.ServeHTTP(w, r.WithContext(audit.WithNamespace(r.Context(), s.endpoint.NamespaceID)))
		})
	}
	return browserMCPCORS(s.runtime.config.MCP.CORS, s.runtime.oauthDiscovery(next))
}
func (s *namespaceRuntimeSource) Reload(ctx context.Context, generation int64) error {
	return s.runtime.Reload(ctx, generation)
}
func (s *namespaceRuntimeSource) Close(ctx context.Context) error { return s.runtime.Close(ctx) }

func (s *namespaceRuntimeSource) RuntimeStatus() (string, int64) {
	return s.runtime.config.Authentication.DefaultMode, int64(s.runtime.manager.Revision())
}

func (s *namespaceRuntimeSource) RuntimeConnection(ctx context.Context) (json.RawMessage, error) {
	folder, uri, err := s.runtime.namespaceConnectionFolder(ctx)
	if err != nil {
		return nil, err
	}
	if folder == nil {
		return nil, fmt.Errorf("namespace connection export is not configured")
	}
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	data, err := fs.ReadFile(folder.FS, strings.TrimPrefix(parsed.Path, "/"))
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}
