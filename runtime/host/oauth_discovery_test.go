package host

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProviderDiscoveryAndChallenge(t *testing.T) {
	service := &Service{config: Config{Authentication: Authentication{PublicMCPURL: "https://mcp.example", Providers: map[string]OAuthProvider{"partners": {Issuer: "https://partners.example"}}}}}
	handler := service.oauthDiscovery(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("unauthenticated request reached MCP") }))
	request := httptest.NewRequest("GET", "http://attacker.example/.well-known/oauth-protected-resource/oauth/partners/mcp", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 200 || !strings.Contains(response.Body.String(), `https://mcp.example/oauth/partners/mcp`) || !strings.Contains(response.Body.String(), `https://partners.example`) {
		t.Fatalf("metadata=%d %s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest("POST", "http://attacker.example/oauth/partners/mcp", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 401 || response.Header().Get("WWW-Authenticate") != `Bearer resource_metadata="https://mcp.example/.well-known/oauth-protected-resource/oauth/partners/mcp"` {
		t.Fatalf("challenge=%d %s", response.Code, response.Header())
	}
}

func TestProviderDiscoveryOnSharedNamespacePath(t *testing.T) {
	prefix := "/namespaces/" + strings.Repeat("a", 64)
	service := &Service{config: Config{NamespaceMCPPath: prefix, Authentication: Authentication{PublicMCPURL: "https://mcp.example", Providers: map[string]OAuthProvider{"partners": {Issuer: "https://partners.example"}}}}}
	handler := service.oauthDiscovery(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("unauthorized MCP reached") }))
	request := httptest.NewRequest("POST", "http://attacker.example/oauth/partners/mcp", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	expected := `Bearer resource_metadata="https://mcp.example/.well-known/oauth-protected-resource` + prefix + `/oauth/partners/mcp"`
	if response.Code != 401 || response.Header().Get("WWW-Authenticate") != expected {
		t.Fatalf("challenge=%d %s", response.Code, response.Header())
	}
	request = httptest.NewRequest("GET", "http://attacker.example/.well-known/oauth-protected-resource/oauth/partners/mcp", nil)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 200 || !strings.Contains(response.Body.String(), "https://mcp.example"+prefix+"/oauth/partners/mcp") {
		t.Fatalf("metadata=%d %s", response.Code, response.Body.String())
	}
}
