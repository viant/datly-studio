package hostconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/internal/bffauth"
	"github.com/viant/scy/auth/jwt"
)

type authTestVerifier struct{}

func (authTestVerifier) VerifyClaims(_ context.Context, token string) (*jwt.Claims, error) {
	if token != "synthetic-access" {
		return nil, fmt.Errorf("invalid token")
	}
	claims := &jwt.Claims{}
	claims.Subject = "owner"
	return claims, nil
}

func TestUppercaseAuthUsesExistingBFFStartCallbackAndApplicationCookies(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/token", r.URL.Path)
		require.NoError(t, r.ParseForm())
		require.Equal(t, "approved", r.Form.Get("code"))
		require.NotEmpty(t, r.Form.Get("code_verifier"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"synthetic-access","token_type":"Bearer","expires_in":300}`)
	}))
	defer provider.Close()
	secretPath := filepath.Join(t.TempDir(), "client.enc")
	client := map[string]any{"ClientID": "synthetic-client", "ClientSecret": "synthetic-secret", "Endpoint": map[string]any{"AuthURL": provider.URL + "/authorize", "TokenURL": provider.URL + "/token"}, "Scopes": []string{"wrong-from-file"}}
	raw, err := json.Marshal(client)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(secretPath, raw, 0600))
	for _, cookieName := range []string{"aistudio_auth", "datastudio_auth"} {
		t.Run(cookieName, func(t *testing.T) {
			raw := base + fmt.Sprintf(`Auth:
  Enabled: true
  CookieName: %s
  OAuth:
    Mode: bff
    Label: Sign in with IdP
    RedirectSameTab: true
    Client:
      ConfigURL: '%s|blowfish://default'
      Scopes: [openid, profile, email, 'plan*']
      WebUIScopes: [web, profile]
      RedirectURIs: ['http://127.0.0.1:8080/v1/studio/auth/callback']
`, cookieName, secretPath)
			config, err := Decode([]byte(raw))
			require.NoError(t, err)
			require.Equal(t, "Sign in with IdP", config.Auth.OAuth.Label)
			require.Nil(t, config.NamespaceConnectionExport, "UI Auth must not derive MCP credentials")
			resolved, err := config.Auth.Resolve(context.Background(), "http://127.0.0.1:8080")
			require.NoError(t, err)
			require.Equal(t, []string{"openid", "profile", "email", "plan*", "web"}, resolved.Scopes)
			resolved.RedirectURL = "http://127.0.0.1:8080/v1/studio/auth/callback"
			sessions, err := bffauth.New(bffauth.Config{CookieName: config.Auth.CookieName}, authTestVerifier{})
			require.NoError(t, err)
			login, err := bffauth.NewLogin(bffauth.LoginConfig{OAuth: *resolved, CookieKey: []byte("0123456789abcdef0123456789abcdef")}, sessions)
			require.NoError(t, err)
			mux := http.NewServeMux()
			sessions.Register(mux)
			require.NoError(t, login.Register(mux))
			start := httptest.NewRecorder()
			mux.ServeHTTP(start, httptest.NewRequest(http.MethodGet, "/v1/studio/auth/login?returnTo="+url.QueryEscape("/?section=window&window=records"), nil))
			require.Equal(t, http.StatusFound, start.Code)
			target, err := url.Parse(start.Header().Get("Location"))
			require.NoError(t, err)
			require.Equal(t, "openid profile email plan* web", target.Query().Get("scope"))
			require.NotEmpty(t, target.Query().Get("code_challenge"))
			cookies := start.Result().Cookies()
			require.Len(t, cookies, 1)
			require.Equal(t, cookieName+"_login_state", cookies[0].Name)
			callback := httptest.NewRequest(http.MethodGet, "/v1/studio/auth/callback?code=approved&state="+url.QueryEscape(target.Query().Get("state")), nil)
			callback.AddCookie(cookies[0])
			completed := httptest.NewRecorder()
			mux.ServeHTTP(completed, callback)
			require.Equal(t, http.StatusSeeOther, completed.Code)
			require.Equal(t, "/?section=window&window=records", completed.Header().Get("Location"))
			found := false
			for _, cookie := range completed.Result().Cookies() {
				if cookie.Name == cookieName {
					found = true
					require.True(t, cookie.HttpOnly)
					require.NotEqual(t, "synthetic-access", cookie.Value)
				}
			}
			require.True(t, found)
		})
	}
}

func TestAuthRejectsAmbiguousOrUnsupportedConfiguration(t *testing.T) {
	valid := `Auth:
  Enabled: true
  CookieName: aistudio_auth
  OAuth:
    Mode: bff
    RedirectSameTab: true
    Client:
      ConfigURL: 'file:///does-not-exist/client.enc|blowfish://default'
      Scopes: [openid]
`
	for _, raw := range []string{
		strings.Replace(valid, "Enabled:", "enabled:", 1), strings.Replace(valid, "Client:", "client:", 1), strings.Replace(valid, "ConfigURL:", "ConfigUrl:", 1), strings.Replace(valid, "Mode: bff", "Mode: implicit", 1), strings.Replace(valid, "RedirectSameTab: true", "RedirectSameTab: false", 1), strings.Replace(valid, "Scopes: [openid]", "Scopes: [profile]", 1), strings.Replace(valid, "Scopes: [openid]", "Scopes: [openid]\n      MobileUIScopes: [mobile]", 1), strings.Replace(valid, "CookieName: aistudio_auth", "CookieName: bad name", 1), valid + "Login: {}\n", valid + "  Unexpected: true\n",
	} {
		_, err := Decode([]byte(base + raw))
		require.Error(t, err, raw)
	}
	config, err := Decode([]byte(base + valid))
	require.NoError(t, err, "decode must not resolve the absent SCY client")
	config.Auth.OAuth.Client.RedirectURIs = []string{"https://other.example/v1/studio/auth/callback"}
	_, err = config.Auth.Resolve(context.Background(), "https://studio.example")
	require.Error(t, err)
	disabled, err := Decode([]byte(base + "Auth:\n  Enabled: false\n"))
	require.NoError(t, err)
	resolved, err := disabled.Auth.Resolve(context.Background(), "https://studio.example")
	require.NoError(t, err)
	require.Nil(t, resolved)
}
