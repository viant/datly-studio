package host

import (
	"context"
	"maps"
	"net/http"

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
		return mcpserver.Config{Source: &namespaceRuntimeSource{runtime}, Implementation: schema.Implementation{Name: "datly-studio-namespace", Version: "1"}, Transport: mcpserver.TransportConfig{Kind: mcpserver.TransportStreamable, Address: endpoint.Address, CORS: config.MCP.CORS}}, nil
	}
}

type namespaceRuntimeSource struct{ runtime *Service }

func (s *namespaceRuntimeSource) Pin(ctx context.Context) (context.Context, mcpserver.ServerService, error) {
	return s.runtime.manager.Pin(ctx)
}
func (s *namespaceRuntimeSource) PinSnapshot(ctx context.Context) (context.Context, mcpserver.ServerService, error) {
	return s.runtime.manager.PinSnapshot(ctx)
}
func (s *namespaceRuntimeSource) WrapHTTP(next http.Handler) http.Handler {
	return browserMCPCORS(s.runtime.config.MCP.CORS, s.runtime.oauthDiscovery(next))
}
func (s *namespaceRuntimeSource) Reload(ctx context.Context, generation int64) error {
	return s.runtime.Reload(ctx, generation)
}
func (s *namespaceRuntimeSource) Close(ctx context.Context) error { return s.runtime.Close(ctx) }

func (s *namespaceRuntimeSource) RuntimeStatus() (string, int64) {
	return s.runtime.config.Authentication.DefaultMode, int64(s.runtime.manager.Revision())
}
