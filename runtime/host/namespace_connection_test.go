package host

import (
	mcpconfig "github.com/viant/mcp/client/config"

	"strings"
	"testing"

	"encoding/json"
)

func TestNamespaceConnectionOAuthUsesSCYAndScopedEndpoint(t *testing.T) {
	for _, prefix := range []string{"", "/namespaces/" + strings.Repeat("a", 64)} {
		config := Config{NamespaceMCPPath: prefix, Authentication: Authentication{DefaultMode: "required", PublicMCPURL: "https://mcp.example", Providers: map[string]OAuthProvider{"employees": {Issuer: "https://idp.example"}}, ConnectionExport: &MCPConnectionExport{Provider: "employees", ClientConfigURL: "file:///var/secrets/studio/oauth-client.enc|blowfish://env/STUDIO_OAUTH_CLIENT_KEY", Scopes: []string{"reports:read"}}}}
		data, err := namespaceConnectionJSON("sales", config)
		if err != nil {
			t.Fatal(err)
		}
		var client mcpconfig.MCPClient
		if err = json.Unmarshal(data, &client); err != nil {
			t.Fatal(err)
		}
		endpoint := "https://mcp.example" + prefix + "/oauth/employees/mcp"
		if client.Name != "sales" || client.Transport.Type != "streamable" || client.Transport.URL != endpoint || client.Auth == nil || client.Auth.Mode != "oauth" || client.Auth.Resource != endpoint {
			t.Fatalf("invalid Agently config: %s", data)
		}
		if client.Auth.InlineProvider.Clients["agently"].ConfigURL != "file:///var/secrets/studio/oauth-client.enc|blowfish://env/STUDIO_OAUTH_CLIENT_KEY" {
			t.Fatal("SCY resource reference lost")
		}
		if client.ProtectedResourceMetadataURL != "https://mcp.example/.well-known/oauth-protected-resource/"+strings.TrimPrefix(prefix+"/oauth/employees/mcp", "/") {
			t.Fatalf("wrong scoped discovery URL: %s", client.ProtectedResourceMetadataURL)
		}
		for _, field := range []string{`"accessToken":`, `"refreshToken":`, `"clientSecret":`, `"encryptionKey":`} {
			if strings.Contains(string(data), field) {
				t.Fatalf("inline secret field: %s", field)
			}
		}
	}
}

func TestNamespaceConnectionDoesNotDowngradeProtectedAuthentication(t *testing.T) {
	config := Config{Authentication: Authentication{DefaultMode: "required", PublicMCPURL: "https://mcp.example"}}
	protected, err := namespaceConnectionJSON("sales", config)
	if err != nil {
		t.Fatal(err)
	}
	var definition mcpconfig.MCPClient
	if err = json.Unmarshal(protected, &definition); err != nil {
		t.Fatal(err)
	}
	if definition.Auth == nil || definition.Auth.BackendForFrontend == nil || !*definition.Auth.BackendForFrontend || !definition.Auth.UseIdToken || definition.Auth.InlineProvider != nil || definition.Auth.ProviderRef != "" || definition.Auth.Mode != "" || definition.ToolsListVisibility != "private" || definition.Transport.URL != "https://mcp.example/mcp" {
		t.Fatalf("protected namespace did not retain consumer BFF authentication: %s", protected)
	}
	for _, value := range []string{"configURL", "clientSecret", "accessToken", "refreshToken", "inlined://"} {
		if strings.Contains(string(protected), value) {
			t.Fatalf("unexpected credential configuration: %s", protected)
		}
	}
	config.Authentication.Providers = map[string]OAuthProvider{"employees": {Issuer: "https://idp.example"}}
	config.Authentication.ConnectionExport = &MCPConnectionExport{Provider: "employees", ClientConfigURL: "inlined://plaintext-credentials"}
	if _, err := namespaceConnectionJSON("sales", config); err == nil {
		t.Fatal("inline credential resource accepted")
	}
	config.Authentication.DefaultMode = "public"
	data, err := namespaceConnectionJSON("sales", config)
	if err != nil {
		t.Fatal(err)
	}
	var client mcpconfig.MCPClient
	if err = json.Unmarshal(data, &client); err != nil {
		t.Fatal(err)
	}
	if client.Auth == nil || client.Auth.ShouldPassUserToken() || client.Auth.BackendForFrontend == nil || *client.Auth.BackendForFrontend || client.Auth.Mode != "" || client.Auth.InlineProvider != nil || client.Transport.URL != "https://mcp.example/mcp" {
		t.Fatal("bare connection inherits authentication or token forwarding")
	}
}
