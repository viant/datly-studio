// Run this local-only acceptance check after starting scale-runtime.yaml.
package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/viant/datly-studio/internal/bffauth"
	"github.com/viant/scy/auth/jwt"
)

type fixtureVerifier struct{}

func (fixtureVerifier) VerifyClaims(context.Context, string) (*jwt.Claims, error) {
	claims := &jwt.Claims{}
	claims.Subject = "scale-review"
	return claims, nil
}

func main() {
	endpoint := os.Getenv("STUDIO_SCALE_RUNTIME_MCP_URL")
	if endpoint == "" {
		endpoint = "http://127.0.0.1:8091"
	}
	target, err := url.Parse(endpoint)
	if err != nil || target == nil || target.Scheme != "http" ||
		(target.Hostname() != "127.0.0.1" && target.Hostname() != "localhost" && target.Hostname() != "::1") {
		fail(fmt.Errorf("scale MCP target must be a loopback HTTP URL: %q", endpoint))
	}
	sessions, err := bffauth.New(bffauth.Config{}, fixtureVerifier{})
	if err != nil {
		fail(err)
	}
	id, _, _, err := sessions.Exchange(context.Background(), "scale-review-local-token")
	if err != nil {
		fail(err)
	}
	proxy, err := sessions.Proxy(target, "/v1/studio/mcp")
	if err != nil {
		fail(err)
	}
	server := httptest.NewServer(proxy)
	defer server.Close()
	unauthorized, err := http.Post(server.URL+"/v1/studio/mcp/mcp", "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	if err != nil {
		fail(err)
	}
	_ = unauthorized.Body.Close()
	if unauthorized.StatusCode != http.StatusUnauthorized {
		fail(fmt.Errorf("cookie-less MCP request returned %d, want 401", unauthorized.StatusCode))
	}
	command := exec.Command("node", "e2e/testdata/verify_scale_mcp.mjs")
	command.Env = append(os.Environ(),
		"STUDIO_SCALE_MCP_URL="+server.URL+"/v1/studio/mcp/mcp",
		"STUDIO_SCALE_MCP_COOKIE="+bffauth.DefaultCookieName+"="+id)
	output, err := command.CombinedOutput()
	if err != nil {
		fail(fmt.Errorf("authenticated scale MCP verifier: %w: %s", err, output))
	}
	fmt.Printf("authenticated BFF MCP: %s", output)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
