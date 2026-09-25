package bffauth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/viant/scy/auth/jwt"
	"golang.org/x/oauth2"
)

type loginVerifier struct{}

func (loginVerifier) VerifyClaims(_ context.Context, token string) (*jwt.Claims, error) {
	if token != "signed-access-token" {
		return nil, errors.New("invalid signature")
	}
	claims := &jwt.Claims{}
	claims.Subject = "owner"
	return claims, nil
}

func TestOAuthLoginCreatesVerifiedServerSideSession(t *testing.T) {
	const redirect = "http://127.0.0.1:8080" + callbackPath
	tokenCalls := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			t.Errorf("unexpected provider path %q", r.URL.Path)
			return
		}
		tokenCalls++
		if err := r.ParseForm(); err != nil {
			t.Error(err)
			return
		}
		code := r.Form.Get("code")
		if code != "approved" && code != "invalid" || r.Form.Get("code_verifier") == "" || r.Form.Get("redirect_uri") != redirect {
			t.Errorf("unexpected token request: %v", r.Form)
		}
		w.Header().Set("Content-Type", "application/json")
		accessToken := "signed-access-token"
		if r.Form.Get("code") == "invalid" {
			accessToken = "invalid-signature"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": accessToken, "token_type": "Bearer", "expires_in": 300})
	}))
	defer provider.Close()
	sessions, err := New(Config{Secure: true}, loginVerifier{})
	if err != nil {
		t.Fatal(err)
	}
	login, err := NewLogin(LoginConfig{OAuth: oauth2.Config{ClientID: "studio-web", RedirectURL: redirect,
		Scopes: []string{"openid", "profile"}, Endpoint: oauth2.Endpoint{AuthURL: provider.URL + "/authorize", TokenURL: provider.URL + "/token"}},
		CookieKey: []byte("0123456789abcdef0123456789abcdef"), Secure: true}, sessions)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	sessions.Register(mux)
	login.Register(mux)
	start := httptest.NewRecorder()
	mux.ServeHTTP(start, httptest.NewRequest(http.MethodGet, loginPath, nil))
	if start.Code != http.StatusFound || start.Header().Get("Referrer-Policy") != "no-referrer" || len(start.Result().Cookies()) != 1 {
		t.Fatalf("start: status=%d headers=%v", start.Code, start.Header())
	}
	authorize, err := url.Parse(start.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	state := authorize.Query().Get("state")
	if authorize.Host != strings.TrimPrefix(provider.URL, "http://") || authorize.Query().Get("response_type") != "code" || authorize.Query().Get("code_challenge_method") != "S256" || authorize.Query().Get("code_challenge") == "" || authorize.Query().Get("redirect_uri") != redirect || state == "" {
		t.Fatalf("unsafe authorization redirect: %s", authorize)
	}
	flowCookie := start.Result().Cookies()[0]
	if flowCookie.Name != stateCookie || !flowCookie.HttpOnly || !flowCookie.Secure || flowCookie.SameSite != http.SameSiteLaxMode || flowCookie.Path != callbackPath {
		t.Fatalf("login state cookie: %+v", flowCookie)
	}
	callback := httptest.NewRequest(http.MethodGet, callbackPath+"?code=approved&state="+url.QueryEscape(state)+"&next=https://attacker.example", nil)
	callback.AddCookie(flowCookie)
	result := httptest.NewRecorder()
	mux.ServeHTTP(result, callback)
	if result.Code != http.StatusSeeOther || result.Header().Get("Location") != "/" || tokenCalls != 1 {
		t.Fatalf("callback: status=%d location=%q body=%q calls=%d", result.Code, result.Header().Get("Location"), result.Body.String(), tokenCalls)
	}
	var sessionCookie *http.Cookie
	for _, cookie := range result.Result().Cookies() {
		if cookie.Name == DefaultCookieName {
			sessionCookie = cookie
		}
	}
	if sessionCookie == nil || !sessionCookie.HttpOnly || !sessionCookie.Secure || sessionCookie.Value == "signed-access-token" {
		t.Fatalf("session cookie: %+v", sessionCookie)
	}
	me := httptest.NewRequest(http.MethodGet, "/v1/studio/auth/me", nil)
	me.AddCookie(sessionCookie)
	meResult := httptest.NewRecorder()
	mux.ServeHTTP(meResult, me)
	if meResult.Code != http.StatusOK || !strings.Contains(meResult.Body.String(), `"subject":"owner"`) {
		t.Fatalf("verified session: %d %s", meResult.Code, meResult.Body.String())
	}

	for name, mutate := range map[string]func(*http.Request){
		"wrong state":     func(r *http.Request) { r.URL.RawQuery = "code=approved&state=wrong" },
		"missing cookie":  func(r *http.Request) { r.Header.Del("Cookie") },
		"tampered cookie": func(r *http.Request) { r.Header.Set("Cookie", stateCookie+"=tampered") },
		"duplicate state": func(r *http.Request) { r.URL.RawQuery += "&state=" + state },
	} {
		t.Run(name, func(t *testing.T) {
			request := callback.Clone(context.Background())
			mutate(request)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized || tokenCalls != 1 {
				t.Fatalf("invalid callback status=%d token calls=%d", response.Code, tokenCalls)
			}
		})
	}
	login.now = func() time.Time { return time.Now().Add(6 * time.Minute) }
	expired := httptest.NewRecorder()
	mux.ServeHTTP(expired, callback)
	if expired.Code != http.StatusUnauthorized || tokenCalls != 1 {
		t.Fatalf("expired state status=%d token calls=%d", expired.Code, tokenCalls)
	}
	login.now = time.Now
	invalidToken := httptest.NewRequest(http.MethodGet, callbackPath+"?code=invalid&state="+url.QueryEscape(state), nil)
	invalidToken.AddCookie(flowCookie)
	invalidResult := httptest.NewRecorder()
	mux.ServeHTTP(invalidResult, invalidToken)
	if invalidResult.Code != http.StatusUnauthorized || tokenCalls != 2 {
		t.Fatalf("unverified token status=%d token calls=%d", invalidResult.Code, tokenCalls)
	}
	for _, cookie := range invalidResult.Result().Cookies() {
		if cookie.Name == DefaultCookieName {
			t.Fatal("unverified token created a BFF session")
		}
	}
}

func TestOAuthLoginRequiresSafeExplicitConfiguration(t *testing.T) {
	sessions, _ := New(Config{}, loginVerifier{})
	base := LoginConfig{OAuth: oauth2.Config{ClientID: "client", RedirectURL: "https://studio.example" + callbackPath,
		Scopes: []string{"openid"}, Endpoint: oauth2.Endpoint{AuthURL: "https://idp.example/authorize", TokenURL: "https://idp.example/token"}},
		CookieKey: []byte("0123456789abcdef0123456789abcdef"), Secure: true}
	if _, err := NewLogin(base, sessions); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*LoginConfig){
		"missing key":       func(c *LoginConfig) { c.CookieKey = nil },
		"missing openid":    func(c *LoginConfig) { c.OAuth.Scopes = []string{"profile"} },
		"insecure provider": func(c *LoginConfig) { c.OAuth.Endpoint.TokenURL = "http://idp.example/token" },
		"wrong callback":    func(c *LoginConfig) { c.OAuth.RedirectURL = "https://studio.example/other" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := base
			mutate(&candidate)
			if _, err := NewLogin(candidate, sessions); err == nil {
				t.Fatal("unsafe login configuration accepted")
			}
		})
	}
}
