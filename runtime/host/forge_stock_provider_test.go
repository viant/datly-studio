package host

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/forge/backend/mcp/portable"
	"github.com/viant/forge/backend/reporting/identity"
	mcpschema "github.com/viant/mcp-protocol/schema"
)

type stockOAuthFixture struct {
	private  ed25519.PrivateKey
	other    ed25519.PrivateKey
	provider *oauth.Provider
}

func newStockOAuthFixture(t *testing.T) *stockOAuthFixture {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	_, other, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := oauth.New(oauth.Config{Issuer: "https://fixture.identity", Audience: "forge-stock-test", Algorithms: []string{"EdDSA"}, Keyfunc: func(*jwtv5.Token) (any, error) { return public, nil }})
	if err != nil {
		t.Fatal(err)
	}
	return &stockOAuthFixture{private: private, other: other, provider: provider}
}

func (f *stockOAuthFixture) token(t *testing.T, key ed25519.PrivateKey, subject string, roles []string, expires time.Time) string {
	t.Helper()
	claims := oauth.Claims{RegisteredClaims: jwtv5.RegisteredClaims{Issuer: "https://fixture.identity", Audience: jwtv5.ClaimStrings{"forge-stock-test"}, Subject: subject,
		ExpiresAt: jwtv5.NewNumericDate(expires), IssuedAt: jwtv5.NewNumericDate(time.Now().Add(-time.Second))}, AccountID: "21", Tenant: "team", Roles: roles, Exposures: []string{}}
	token, err := jwtv5.NewWithClaims(jwtv5.SigningMethodEdDSA, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func TestStockForgeProviderUsesExplicitWindowAndComponentBindings(t *testing.T) {
	identityFixture := newStockOAuthFixture(t)
	validToken := identityFixture.token(t, identityFixture.private, "alice", []string{"reader"}, time.Now().Add(time.Minute))
	invalidSignature := identityFixture.token(t, identityFixture.other, "alice", []string{"reader"}, time.Now().Add(time.Minute))
	expiredToken := identityFixture.token(t, identityFixture.private, "alice", []string{"reader"}, time.Now().Add(-time.Minute))
	roleDeniedToken := identityFixture.token(t, identityFixture.private, "alice", []string{"visitor"}, time.Now().Add(time.Minute))
	var definitionPath string
	var fixture *portable.Definition
	window := []byte(`window:
  view:
    content:
      id: root
      title: Overview
dataSources:
  records:
    id: records
    selectors:
      data: Records
    cardinality: many
    parameters: []
`)
	component := ComponentReference{Kind: "dynamic", ID: "records", Revision: "1", Method: http.MethodGet, Route: "/records"}
	windowPolicy := func(action string) authz.Policy {
		return authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}
	}
	definition := func() authz.Document {
		return authz.Document{Resource: authz.Resource{Kind: "window", ID: "team/overview", Version: identity.WorkingCandidate, Tenant: "*"}, Revision: 1,
			Policies: map[string]authz.Policy{"discover": windowPolicy("discover"), "describe": windowPolicy("describe"), "execute": windowPolicy("execute")}}
	}
	selection := authz.SelectionDocument{Resource: authz.ResourceFamily{Kind: "window", ID: "team/overview", Tenant: "*"}, Revision: 1, DefaultVersion: identity.WorkingCandidate}
	windowConfig := ForgeWindow{URI: "window://team/overview", DefinitionPath: "window.yaml", Components: map[string]ComponentReference{"records": component}}
	testDynamicHost(t, true, false, dynamicHostExtension{
		configure: func(config *Config) {
			config.Access = &ResourceAccessConfig{Tenant: "*", Provider: identityFixture.provider}
			config.Forge = &ForgeConfig{Windows: []ForgeWindow{windowConfig}, Policies: []authz.Document{definition()}, Selections: []authz.SelectionDocument{selection}}
			definitionPath = filepath.Join(config.RootDir, windowConfig.DefinitionPath)
			if err := os.WriteFile(definitionPath, window, 0600); err != nil {
				t.Fatal(err)
			}
		},
		verify: func(host *Service, mcpAddress string) {
			// The configured static provider owns both the window actions and exact
			// component revision checks. No request value supplies the provider.
			windowDocument := definition()
			windowDocument.Resource.Tenant = "*"
			policies, err := authz.NewStaticStore([]authz.Document{windowDocument, {
				Resource: authz.Resource{Kind: "component", ID: "records", Version: "1", Tenant: "*"}, Revision: 1,
				Policies: map[string]authz.Policy{"describe": {Mode: "public"}, "execute": {Mode: "public"}},
			}})
			if err != nil {
				t.Fatal(err)
			}
			host.resourceAccess.Store = policies
			token := validToken
			testCtx := oauth.WithBearer(context.Background(), token)
			if _, err := host.config.ForgeProvider.Host.Catalog(testCtx, &portable.CatalogInput{ContractVersion: portable.Version}); err != nil {
				t.Fatalf("stock host catalog: %v", err)
			}
			resolver, err := host.config.ForgeProvider.ResourceResolver(testCtx)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := resolver.Resolve(testCtx, identity.ResourceRef{URI: "window://team/overview"}); err != nil {
				t.Fatalf("stock window resolve: %v", err)
			}
			call := func(method string, params any) string {
				t.Helper()
				if method == "resources/read" {
					params.(map[string]any)["_meta"] = map[string]any{
						"io.modelcontextprotocol/protocolVersion":    mcpschema.LatestProtocolVersion,
						"io.modelcontextprotocol/clientCapabilities": map[string]any{},
					}
				}
				body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
				if err != nil {
					t.Fatal(err)
				}
				request, err := http.NewRequest(http.MethodPost, "http://"+mcpAddress+"/forge/mcp", bytes.NewReader(body))
				if err != nil {
					t.Fatal(err)
				}
				request.Header.Set("Authorization", "Bearer "+token)
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Accept", "application/json, text/event-stream")
				request.Header.Set(mcpschema.HeaderProtocolVersion, mcpschema.LatestProtocolVersion)
				request.Header.Set(mcpschema.HeaderMethod, method)
				if method == "tools/call" {
					var tool struct {
						Name string `json:"name"`
					}
					if err := json.Unmarshal(mustJSON(t, params), &tool); err != nil {
						t.Fatal(err)
					}
					request.Header.Set(mcpschema.HeaderName, tool.Name)
				} else if method == "resources/read" {
					request.Header.Set(mcpschema.HeaderName, params.(map[string]any)["uri"].(string))
				}
				response, err := http.DefaultClient.Do(request)
				if err != nil {
					t.Fatal(err)
				}
				defer response.Body.Close()
				data, err := io.ReadAll(response.Body)
				if err != nil || response.StatusCode != http.StatusOK {
					t.Fatalf("MCP %s status=%d err=%v body=%s", method, response.StatusCode, err, data)
				}
				return string(data)
			}
			callTool := func(name string, arguments map[string]any) string {
				return call("tools/call", map[string]any{"name": name, "arguments": arguments, "_meta": map[string]any{
					"io.modelcontextprotocol/protocolVersion":    mcpschema.LatestProtocolVersion,
					"io.modelcontextprotocol/clientCapabilities": map[string]any{},
				}})
			}
			catalog := callTool(portable.CatalogTool, map[string]any{"contractVersion": portable.Version, "limit": 20})
			if !strings.Contains(catalog, "window://team/overview") || !strings.Contains(catalog, "Overview") {
				t.Fatalf("catalog did not expose authorized window: %s", catalog)
			}
			definitionResult := callTool(portable.DefinitionTool, map[string]any{"contractVersion": portable.Version, "windowKey": "team_overview"})
			if !strings.Contains(definitionResult, "records") || !strings.Contains(definitionResult, "contentFingerprint") {
				t.Fatalf("definition omitted exact component pin: %s", definitionResult)
			}
			read := call("resources/read", map[string]any{"uri": "window://team/overview"})
			if !strings.Contains(read, "Overview") || !strings.Contains(read, "working") {
				t.Fatalf("resources/read did not return the approved working window: %s", read)
			}
			requestCtx := oauth.WithBearer(context.Background(), validToken)
			fixture, err = host.config.ForgeProvider.Definition(requestCtx, &portable.DefinitionInput{ContractVersion: portable.Version, Resource: &identity.ResourceRef{URI: "window://team/overview"}})
			if err != nil {
				t.Fatal(err)
			}
			fetch := func(pin *portable.Definition) string {
				t.Helper()
				return callTool(portable.FetchTool, map[string]any{"contractVersion": portable.Version, "resource": pin.Resource,
					"windowKey": pin.Window.WindowKey, "dataSourceId": "records", "definitionRevision": pin.DefinitionRevision, "inputs": map[string]any{}})
			}
			good := fetch(fixture)
			if !strings.Contains(good, "ready") || !strings.Contains(good, "primary") {
				t.Fatalf("stock provider did not execute the pinned native component: %s", good)
			}
			for name, rejected := range map[string]string{"invalid signature": invalidSignature, "expired token": expiredToken, "role denied": roleDeniedToken} {
				token = rejected
				denied := callTool(portable.DefinitionTool, map[string]any{"contractVersion": portable.Version, "windowKey": "team_overview"})
				if !strings.Contains(denied, "isError") || strings.Contains(denied, "contentFingerprint") {
					t.Fatalf("%s identity was not denied by the Forge route: %s", name, denied)
				}
			}
			token = validToken
			exact, err := host.exactDefinition(requestCtx, "records", 1)
			if err != nil {
				t.Fatal(err)
			}
			changedDQL := strings.Replace(exact.dql, "SELECT id,name FROM records", "SELECT id,'changed' AS name FROM records", 1)
			if changedDQL == exact.dql {
				t.Fatal("component SQL fixture did not contain its expected source")
			}
			if _, err := host.studio.ExecContext(requestCtx, "UPDATE report_versions SET generated_dql=?, authored_dql=? WHERE report_id='records' AND version_no=1", changedDQL, changedDQL); err != nil {
				t.Fatal(err)
			}
			if stale := fetch(fixture); strings.Contains(stale, "ready") || !strings.Contains(stale, "isError") {
				t.Fatalf("stale SQL binding executed: %s", stale)
			}
			fresh, err := host.config.ForgeProvider.Definition(requestCtx, &portable.DefinitionInput{ContractVersion: portable.Version, Resource: &identity.ResourceRef{URI: "window://team/overview"}})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(definitionPath, bytes.Replace(window, []byte("Overview"), []byte("Changed"), 1), 0600); err != nil {
				t.Fatal(err)
			}
			if stale := fetch(fresh); strings.Contains(stale, "ready") || !strings.Contains(stale, "isError") {
				t.Fatalf("stale window content executed: %s", stale)
			}
		},
	})
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestStockForgeConfigRequiresExplicitPolicyAndSelection(t *testing.T) {
	identityFixture := newStockOAuthFixture(t)
	config := &Config{Access: &ResourceAccessConfig{Tenant: "*", Provider: identityFixture.provider}, Forge: &ForgeConfig{
		Windows: []ForgeWindow{{URI: "window://team/overview", DefinitionPath: "window.yaml", Components: map[string]ComponentReference{"records": {Kind: "dynamic", ID: "records", Revision: "1", Method: http.MethodGet, Route: "/records"}}}},
		Policies: []authz.Document{{Resource: authz.Resource{Kind: "window", ID: "team/overview", Version: identity.WorkingCandidate, Tenant: "*"}, Revision: 1,
			Policies: map[string]authz.Policy{"discover": {Mode: "public"}, "describe": {Mode: "public"}, "execute": {Mode: "public"}}}},
	}}
	config.HTTP.Address, config.MCP.Address = "127.0.0.1:0", "127.0.0.1:0"
	config.Studio = Studio{Driver: "sqlite", DSN: "file:forge-config"}
	config.Admin.Token, config.Authentication.DefaultMode = "test", "public"
	if err := config.Validate(); err == nil {
		t.Fatal("stock Forge configuration without an explicit selection was accepted")
	}
}
