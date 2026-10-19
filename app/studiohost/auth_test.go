package studiohost_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
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
}
