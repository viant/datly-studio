package host

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/viant/datly-studio/runtime/namespacemcp"
	mcpresource "github.com/viant/datly/mcp/resource"
	"net/url"
	"strings"
	"testing/fstest"

	agentlyconfig "github.com/viant/agently-core/protocol/mcp/config"
	"github.com/viant/mcp"
	authconfig "github.com/viant/mcp/client/auth/config"
)

// namespaceConnectionJSON serializes configuration references only. Token
// resolution and encryption belong to Agently and the configured SCY resource.
func namespaceConnectionJSON(name string, config Config) ([]byte, error) {
	base := strings.TrimRight(config.Authentication.PublicMCPURL, "/")
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("namespace MCP public URL is invalid")
	}
	endpoint := base + config.NamespaceMCPPath + "/mcp"
	type transport struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	}
	document := struct {
		Name                         string          `json:"name"`
		Transport                    transport       `json:"transport"`
		Auth                         *mcp.ClientAuth `json:"auth,omitempty"`
		ToolsListVisibility          string          `json:"toolsListVisibility"`
		ProtectedResourceMetadataURL string          `json:"protectedResourceMetadataURL,omitempty"`
	}{Name: name, Transport: transport{Type: "streamable", URL: endpoint}, ToolsListVisibility: "private"}
	// An omitted auth block inherits the consumer's legacy token-forwarding
	// defaults. Bare means explicitly no BFF and no user-token forwarding.
	disabled := false
	document.Auth = &mcp.ClientAuth{BackendForFrontend: &disabled, PassUserToken: &disabled}
	if config.Authentication.DefaultMode != "public" {
		export := config.Authentication.ConnectionExport
		if export == nil {
			return nil, fmt.Errorf("namespace OAuth connection export is not configured")
		}
		provider, ok := config.Authentication.Providers[export.Provider]
		if !ok || export.Provider == "" {
			return nil, fmt.Errorf("namespace OAuth provider is not configured")
		}
		// SCY accepts both ordinary resource URLs and URL|key-reference forms.
		// Preserve the reference; exporting configuration must never fetch it.
		parts := strings.SplitN(export.ClientConfigURL, "|", 2)
		secret, err := url.Parse(parts[0])
		if err != nil || strings.TrimSpace(parts[0]) == "" || strings.HasPrefix(parts[0], "inlined:") || secret.User != nil || secret.Fragment != "" {
			return nil, fmt.Errorf("OAuth client configuration requires a SCY secret resource")
		}
		if len(parts) == 2 {
			key, keyErr := url.Parse(parts[1])
			if keyErr != nil || key.Scheme == "" || key.Scheme == "inlined" || key.User != nil {
				return nil, fmt.Errorf("SCY encryption must use a key resource reference")
			}
		}
		providerPath := "/oauth/" + url.PathEscape(export.Provider) + "/mcp"
		document.Transport.URL = base + config.NamespaceMCPPath + providerPath
		document.ProtectedResourceMetadataURL = base + "/.well-known/oauth-protected-resource/" + strings.TrimPrefix(config.NamespaceMCPPath+providerPath, "/")
		document.Auth = &mcp.ClientAuth{Mode: "oauth", ClientRef: "agently", Resource: document.Transport.URL, TokenType: "accessToken", Resolution: "eager", WorkspaceTokenReuse: "never",
			Scopes: append([]string(nil), export.Scopes...), InlineProvider: &authconfig.OAuthProvider{ID: export.Provider, Issuer: provider.Issuer, DefaultClient: "agently", Clients: map[string]*authconfig.OAuthClient{"agently": {ConfigURL: export.ClientConfigURL, RedirectURI: export.RedirectURI}}}}
		if err := document.Auth.InlineProvider.Validate(); err != nil {
			return nil, fmt.Errorf("namespace OAuth provider is invalid: %w", err)
		}
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, err
	}
	// Round-trip through the actual consumer contract, not a Studio-only schema.
	var consumer agentlyconfig.MCPClient
	if err = json.Unmarshal(data, &consumer); err != nil {
		return nil, err
	}
	if consumer.ClientOptions == nil || consumer.Transport.URL != document.Transport.URL {
		return nil, fmt.Errorf("invalid Agently MCP configuration")
	}
	return data, nil
}

func (s *Service) namespaceConnectionFolder(ctx context.Context) (*mcpresource.Folder, string, error) {
	if s.config.NamespaceID == "" || (s.config.Authentication.PublicMCPURL == "" && s.config.Authentication.ConnectionExport == nil) || (s.config.Authentication.DefaultMode != "public" && s.config.Authentication.ConnectionExport == nil) {
		return nil, "", nil
	}
	rows, err := (namespacemcp.SQLDefinitions{DB: s.studio}).LoadNamespace(ctx, s.config.NamespaceID)
	if err != nil {
		return nil, "", err
	}
	if len(rows) != 1 || rows[0].Name == "" || !rows[0].Active {
		return nil, "", fmt.Errorf("namespace connection is unavailable")
	}
	name := rows[0].Name
	data, err := namespaceConnectionJSON(name, s.config)
	if err != nil {
		return nil, "", err
	}
	return &mcpresource.Folder{Namespace: "studio_namespace_connection", Root: ".", URIPrefix: "mcp://ns", Description: "Agently namespace MCP connection (JSON)", FS: fstest.MapFS{name: &fstest.MapFile{Data: data, Mode: 0444}}}, "mcp://ns/" + url.PathEscape(name), nil
}
