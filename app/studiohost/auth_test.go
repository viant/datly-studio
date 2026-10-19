package studiohost_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/app/hostconfig"
	"github.com/viant/datly-studio/app/studioapi"
	"github.com/viant/datly-studio/app/studiohost"
	mcpschema "github.com/viant/mcp-protocol/schema"
	mcpcfg "github.com/viant/mcp/client/config"
)

func TestUppercaseAuthHostWithoutJWTClaims(t *testing.T) {
	if os.Getenv("STUDIO_AUTH_CONFIG_TEST") == "" {
		command := exec.Command(os.Args[0], "-test.run=^TestUppercaseAuthHostWithoutJWTClaims$", "-test.v")
		command.Env = append(os.Environ(), "STUDIO_AUTH_CONFIG_TEST=1")
		out, err := command.CombinedOutput()
		require.NoError(t, err, string(out))
		return
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	root := t.TempDir()
	public, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	publicPath := filepath.Join(root, "public.pem")
	require.NoError(t, os.WriteFile(publicPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: public}), 0600))
	mode := "valid"
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/token", r.URL.Path)
		require.NoError(t, r.ParseForm())
		require.NotEmpty(t, r.Form.Get("code_verifier"))
		claims := jwtlib.MapClaims{"iss": "https://signed-issuer.example/", "aud": "studio-client", "sub": "owner", "account_id": 42, "membershipGroups": []string{"owners"}, "exp": time.Now().Add(time.Minute).Unix()}
		if mode == "expired" {
			claims["exp"] = time.Now().Add(-time.Minute).Unix()
		}
		if mode == "no-exp" {
			delete(claims, "exp")
		}
		if mode == "no-sub" {
			delete(claims, "sub")
		}
		if mode == "wrong-audience" {
			claims["aud"] = "other-client"
		}
		token, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims).SignedString(key)
		require.NoError(t, err)
		if mode == "tampered" {
			token = token[:len(token)-10] + "tamperedxx"
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"access_token": token, "token_type": "Bearer", "expires_in": 60}))
	}))
	defer provider.Close()
	clientPath := filepath.Join(root, "client.enc")
	raw, err := json.Marshal(map[string]any{"ClientID": "studio-client", "ClientSecret": "synthetic-secret", "Endpoint": map[string]string{"AuthURL": provider.URL + "/authorize", "TokenURL": provider.URL + "/token"}, "Scopes": []string{"openid"}})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(clientPath, raw, 0600))
	document := map[string]any{"BaseDir": root, "Endpoint": map[string]any{"Address": "127.0.0.1:0"}, "MCP": map[string]any{"Address": "127.0.0.1:0"}, "NamespaceMCP": map[string]any{"Address": "127.0.0.1:0"}, "Store": map[string]any{"Driver": "sqlite", "DSN": filepath.Join(root, "studio.db")}, "Tenant": "server-tenant", "JWTValidator": map[string]any{"RSA": []any{map[string]any{"URL": publicPath}}}, "Auth": map[string]any{"Enabled": true, "CookieName": "aistudio_test", "OAuth": map[string]any{"Mode": "bff", "Label": "Sign in with IdP", "RedirectSameTab": true, "Client": map[string]any{"ConfigURL": clientPath + "|blowfish://default", "Scopes": []string{"openid", "profile", "email", "plan*"}}}}}
	raw, err = json.Marshal(document)
	require.NoError(t, err)
	config, err := hostconfig.Decode(raw)
	require.NoError(t, err)
	require.Nil(t, config.JWTClaims)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan studiohost.Endpoints, 1)
	done := make(chan error, 1)
	go func() {
		done <- studiohost.Serve(ctx, config, studiohost.Options{Diagnostics: io.Discard, Browser: studioapi.Options{StaticFS: fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("UI")}}}, Ready: func(endpoints studiohost.Endpoints) { ready <- endpoints }})
	}()
	var endpoints studiohost.Endpoints
	select {
	case endpoints = <-ready:
	case err := <-done:
		t.Fatalf("startup: %v", err)
	case <-time.After(30 * time.Second):
		t.Fatal("startup timeout")
	}
	defer func() { cancel(); require.NoError(t, <-done) }()
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{Jar: jar, Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	origin := "http://" + endpoints.HTTP
	for _, testMode := range []string{"valid", "expired", "no-exp", "no-sub", "wrong-audience", "tampered"} {
		t.Run(testMode, func(t *testing.T) {
			mode = testMode
			start, err := client.Get(origin + "/v1/studio/auth/login?returnTo=" + url.QueryEscape("/?section=window"))
			require.NoError(t, err)
			start.Body.Close()
			require.Equal(t, http.StatusFound, start.StatusCode)
			target, err := url.Parse(start.Header.Get("Location"))
			require.NoError(t, err)
			require.Equal(t, "openid profile email plan*", target.Query().Get("scope"))
			callback := origin + "/v1/studio/auth/callback?code=approved&state=" + url.QueryEscape(target.Query().Get("state"))
			completed, err := client.Get(callback)
			require.NoError(t, err)
			defer completed.Body.Close()
			if testMode != "valid" {
				require.Equal(t, http.StatusUnauthorized, completed.StatusCode)
				return
			}
			require.Equal(t, http.StatusSeeOther, completed.StatusCode)
			require.Equal(t, "/?section=window", completed.Header.Get("Location"))
			found := false
			for _, cookie := range completed.Cookies() {
				if cookie.Name == "aistudio_test" {
					found = true
					require.True(t, cookie.HttpOnly)
				}
			}
			require.True(t, found)
			me, err := client.Get(origin + "/v1/studio/auth/me")
			require.NoError(t, err)
			defer me.Body.Close()
			body, err := io.ReadAll(me.Body)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, me.StatusCode)
			require.Contains(t, string(body), "owner")
			require.NotContains(t, string(body), "synthetic-secret")
			browser, err := client.Get(origin + "/studio-config.json")
			require.NoError(t, err)
			defer browser.Body.Close()
			body, err = io.ReadAll(browser.Body)
			require.NoError(t, err)
			require.True(t, strings.Contains(string(body), "Sign in with IdP"))
		})
	}
	// No namespace OAuth export settings are configured. The protected native
	// namespace must nevertheless publish its existing consumer-BFF reference.
	database, err := sql.Open("sqlite", filepath.Join(root, "studio.db"))
	require.NoError(t, err)
	defer database.Close()
	namespaceID := strings.Repeat("a", 64)
	_, err = database.Exec(`INSERT INTO namespaces(namespace_id,name,title,owner_id,status,mcp_enabled,mcp_port,etag,created_at,updated_at) VALUES(?, 'bff-namespace','BFF namespace','owner','active',1,0,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, namespaceID)
	require.NoError(t, err)
	bearer, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{"iss": "https://signed-issuer.example/", "aud": "studio-client", "sub": "owner", "account_id": 42, "exp": time.Now().Add(time.Minute).Unix()}).SignedString(key)
	require.NoError(t, err)
	endpoint := "http://" + endpoints.NamespaceMCP + "/namespaces/" + namespaceID + "/mcp"
	call := func(method string, params map[string]any, credential string) (int, map[string]json.RawMessage) {
		params["_meta"] = map[string]any{"io.modelcontextprotocol/protocolVersion": mcpschema.LatestProtocolVersion, "io.modelcontextprotocol/clientCapabilities": map[string]any{}}
		raw, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params})
		require.NoError(t, err)
		request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(string(raw)))
		require.NoError(t, err)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json")
		request.Header.Set(mcpschema.HeaderMethod, method)
		request.Header.Set("Mcp-Protocol-Version", mcpschema.LatestProtocolVersion)
		if uri, ok := params["uri"].(string); ok {
			request.Header.Set(mcpschema.HeaderName, uri)
		}
		if credential != "" {
			request.Header.Set("Authorization", "Bearer "+credential)
		}
		response, err := client.Do(request)
		require.NoError(t, err)
		defer response.Body.Close()
		payload, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		result := map[string]json.RawMessage{}
		_ = json.Unmarshal(payload, &result)
		return response.StatusCode, result
	}
	var inventory mcpschema.ListResourcesResult
	require.Eventually(t, func() bool {
		status, reply := call("resources/list", map[string]any{}, bearer)
		if status != http.StatusOK || reply["error"] != nil {
			return false
		}
		if json.Unmarshal(reply["result"], &inventory) != nil {
			return false
		}
		return len(inventory.Resources) == 1
	}, 20*time.Second, 100*time.Millisecond)
	uri := inventory.Resources[0].Uri
	require.Equal(t, "mcp://ns/bff-namespace", uri)
	status, reply := call("resources/read", map[string]any{"uri": uri}, bearer)
	require.Equal(t, http.StatusOK, status)
	require.Nil(t, reply["error"])
	var read mcpschema.ReadResourceResult
	require.NoError(t, json.Unmarshal(reply["result"], &read))
	require.Len(t, read.Contents, 1)
	var definition mcpcfg.MCPClient
	require.NoError(t, json.Unmarshal([]byte(read.Contents[0].Text), &definition))
	require.Equal(t, endpoint, definition.Transport.URL)
	require.Equal(t, "private", definition.ToolsListVisibility)
	require.NotNil(t, definition.Auth)
	require.NotNil(t, definition.Auth.BackendForFrontend)
	require.True(t, *definition.Auth.BackendForFrontend)
	require.True(t, definition.Auth.UseIdToken)
	require.Nil(t, definition.Auth.InlineProvider)
	for _, method := range []string{"resources/list", "resources/read"} {
		params := map[string]any{}
		if method == "resources/read" {
			params["uri"] = uri
		}
		status, reply = call(method, params, "")
		denied := status == http.StatusUnauthorized || status == http.StatusForbidden || reply["error"] != nil
		if method == "resources/list" && !denied {
			// Native MCP discovery denies individual private resources by
			// omission; it need not reject an otherwise empty inventory.
			var anonymous mcpschema.ListResourcesResult
			require.NoError(t, json.Unmarshal(reply["result"], &anonymous))
			require.Empty(t, anonymous.Resources, "unauthenticated inventory must not expose the protected BFF reference")
		} else {
			require.True(t, denied, "unauthenticated native connection %s must fail", method)
		}
	}

}
