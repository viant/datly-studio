package bffauth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/viant/datly-studio/store/sql/migrate"
	"github.com/viant/scy/auth/jwt"
	"golang.org/x/oauth2"
	_ "modernc.org/sqlite"
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

type identityVerifier struct{}

func (identityVerifier) VerifyClaims(_ context.Context, token string) (*jwt.Claims, error) {
	claims := &jwt.Claims{}
	claims.Subject = "owner"
	claims.Issuer = "https://identity.example"
	claims.Audience = jwtv5.ClaimStrings{"studio-web"}
	switch token {
	case "id-1":
		claims.ExpiresAt = jwtv5.NewNumericDate(time.Now().Add(30 * time.Second))
	case "id-2":
		claims.ExpiresAt = jwtv5.NewNumericDate(time.Now().Add(time.Hour))
	case "id-long":
		claims.ExpiresAt = jwtv5.NewNumericDate(time.Now().Add(2 * time.Hour))
	default:
		return nil, errors.New("invalid signature")
	}
	return claims, nil
}

func TestBrowserIdentitySessionRefreshesServerSideAndReturnsOnlyIDToken(t *testing.T) {
	const origin = "https://studio.example"
	const sessionCookieName = "studio_custom"
	refreshCalls := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Form.Get("grant_type") {
		case "authorization_code":
			if r.Form.Get("code") != "approved" || r.Form.Get("code_verifier") == "" {
				t.Errorf("unexpected code exchange: %v", r.Form)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "access-1", "refresh_token": "refresh-1",
				"id_token": "id-1", "token_type": "Bearer", "expires_in": 3600})
		case "refresh_token":
			refreshCalls++
			if r.Form.Get("refresh_token") != "refresh-1" {
				t.Errorf("unexpected refresh request: %v", r.Form)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "access-2", "refresh_token": "refresh-2",
				"id_token": "id-2", "token_type": "Bearer", "expires_in": 3600})
		default:
			t.Errorf("unexpected grant type: %v", r.Form)
		}
	}))
	defer provider.Close()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	migration, err := migrate.New()
	if err != nil {
		t.Fatal(err)
	}
	if err := migration.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	store, err := NewSQLStore(db, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close(context.Background())
	sessions, err := New(Config{Issuer: "https://identity.example", Audience: "studio-web", Secure: true,
		CookieName: sessionCookieName, Store: store}, identityVerifier{})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := sessions.ExchangeIDToken(context.Background(), "id-long", "refresh-token"); err == nil {
		t.Fatal("ID token beyond the 60-minute policy was accepted")
	}
	if _, _, _, err := sessions.ExchangeIDToken(context.Background(), "id-2", "refresh-token"); err != nil {
		t.Fatalf("60-minute ID token was rejected: %v", err)
	}
	login, err := NewLogin(LoginConfig{OAuth: oauth2.Config{ClientID: "studio-web", ClientSecret: "backend-only",
		RedirectURL: origin + callbackPath, Scopes: []string{"openid"},
		Endpoint: oauth2.Endpoint{AuthURL: provider.URL + "/authorize", TokenURL: provider.URL + "/token"}},
		CookieKey: []byte("0123456789abcdef0123456789abcdef"), Secure: true,
		UseIDToken: true, AllowedOrigin: origin}, sessions)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := login.DatlyHTTP()
	if err != nil {
		t.Fatal(err)
	}
	start := httptest.NewRecorder()
	mux.ServeHTTP(start, httptest.NewRequest(http.MethodGet, loginPath, nil))
	if start.Code != http.StatusFound || len(start.Result().Cookies()) != 1 {
		t.Fatalf("Datly login status=%d headers=%v", start.Code, start.Header())
	}
	authorize, err := url.Parse(start.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if authorize.Query().Get("code_challenge_method") != "S256" || authorize.Query().Get("code_challenge") == "" {
		t.Fatalf("missing PKCE: %s", authorize)
	}
	for _, badQuery := range []string{
		"code=approved&state=" + url.QueryEscape(authorize.Query().Get("state")) + "&state=duplicate",
		"code=approved&state=wrong",
		"error=access_denied",
	} {
		bad := httptest.NewRequest(http.MethodGet, callbackPath+"?"+badQuery, nil)
		bad.AddCookie(start.Result().Cookies()[0])
		denied := httptest.NewRecorder()
		mux.ServeHTTP(denied, bad)
		if denied.Code != http.StatusUnauthorized || len(denied.Header().Values("Set-Cookie")) != 1 {
			t.Fatalf("invalid Datly callback status=%d headers=%v", denied.Code, denied.Header())
		}
	}
	callback := httptest.NewRequest(http.MethodGet, callbackPath+"?code=approved&state="+url.QueryEscape(authorize.Query().Get("state")), nil)
	callback.AddCookie(start.Result().Cookies()[0])
	completed := httptest.NewRecorder()
	mux.ServeHTTP(completed, callback)
	if completed.Code != http.StatusSeeOther || completed.Header().Get("Location") != "/" {
		t.Fatalf("Datly callback status=%d body=%q headers=%v", completed.Code, completed.Body.String(), completed.Header())
	}
	id := ""
	for _, cookie := range completed.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			id = cookie.Value
		}
	}
	if id == "" || id == "id-1" || id == "refresh-1" {
		t.Fatalf("unsafe opaque cookie: %q", id)
	}
	initial, found, err := sessions.store.Get(context.Background(), id)
	if err != nil || !found || initial.token != "id-1" || initial.refreshToken != "refresh-1" || time.Until(initial.expiresAt) < time.Hour {
		t.Fatalf("Datly callback session found=%v err=%v value=%+v", found, err, initial)
	}
	var ciphertext []byte
	if err := db.QueryRow(`SELECT payload_ciphertext FROM bff_sessions WHERE session_id_hash=?`, sessionHash(id)).Scan(&ciphertext); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ciphertext, []byte("id-1")) || bytes.Contains(ciphertext, []byte("refresh-1")) {
		t.Fatal("Datly auth stored token material in plaintext")
	}
	request := func(requestOrigin string) *http.Request {
		r := httptest.NewRequest(http.MethodPost, tokenPath, nil)
		r.Header.Set("Origin", requestOrigin)
		r.AddCookie(&http.Cookie{Name: sessionCookieName, Value: id})
		return r
	}
	forbidden := httptest.NewRecorder()
	mux.ServeHTTP(forbidden, request("https://attacker.example"))
	if forbidden.Code != http.StatusForbidden || refreshCalls != 0 {
		t.Fatalf("cross-origin status=%d calls=%d", forbidden.Code, refreshCalls)
	}
	result := httptest.NewRecorder()
	mux.ServeHTTP(result, request(origin))
	if result.Code != http.StatusOK || refreshCalls != 1 {
		t.Fatalf("refresh status=%d body=%q calls=%d", result.Code, result.Body.String(), refreshCalls)
	}
	var body map[string]any
	if err := json.Unmarshal(result.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["id_token"] != "id-2" || body["subject"] != "owner" || body["refresh_token"] != nil || body["access_token"] != nil {
		t.Fatalf("unsafe browser response: %v", body)
	}
	stored, ok, err := sessions.store.Get(context.Background(), id)
	if err != nil || !ok || stored.refreshToken != "refresh-2" || stored.token != "id-2" {
		t.Fatalf("rotated session=%+v found=%v err=%v", stored, ok, err)
	}
	again := httptest.NewRecorder()
	mux.ServeHTTP(again, request(origin))
	if again.Code != http.StatusOK || refreshCalls != 1 {
		t.Fatalf("unnecessary refresh status=%d calls=%d", again.Code, refreshCalls)
	}
	foreignLogout := httptest.NewRequest(http.MethodDelete, "/v1/studio/auth/session", nil)
	foreignLogout.Header.Set("Origin", "https://attacker.example")
	foreignLogout.AddCookie(&http.Cookie{Name: sessionCookieName, Value: id})
	foreignResult := httptest.NewRecorder()
	mux.ServeHTTP(foreignResult, foreignLogout)
	if foreignResult.Code != http.StatusForbidden {
		t.Fatalf("foreign logout status=%d", foreignResult.Code)
	}
	logout := httptest.NewRequest(http.MethodDelete, "/v1/studio/auth/session", nil)
	logout.Header.Set("Origin", origin)
	logout.AddCookie(&http.Cookie{Name: sessionCookieName, Value: id})
	loggedOut := httptest.NewRecorder()
	mux.ServeHTTP(loggedOut, logout)
	if loggedOut.Code != http.StatusNoContent || len(loggedOut.Result().Cookies()) != 1 || loggedOut.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatalf("Datly logout status=%d cookies=%v", loggedOut.Code, loggedOut.Result().Cookies())
	}
	missing := httptest.NewRecorder()
	mux.ServeHTTP(missing, request(origin))
	if missing.Code != http.StatusUnauthorized {
		t.Fatalf("logged-out token status=%d", missing.Code)
	}
	for _, target := range []struct{ method, path string }{
		{http.MethodGet, "/v1/studio/auth/me"},
		{http.MethodPost, "/v1/studio/auth/session"},
		{http.MethodPost, "/v1/studio/sdk/components.list"},
	} {
		result := httptest.NewRecorder()
		mux.ServeHTTP(result, httptest.NewRequest(target.method, target.path, nil))
		if result.Code != http.StatusNotFound && result.Code != http.StatusMethodNotAllowed {
			t.Fatalf("unexpected auth-only route %s %s status=%d", target.method, target.path, result.Code)
		}
	}
}

func TestBrowserIdentityRefreshOnceAcrossStudioInstances(t *testing.T) {
	for _, candidate := range []struct {
		name, refreshToken string
	}{
		{"rotating refresh token", "refresh-2"},
		{"unchanged refresh token", "refresh-1"},
		{"omitted refresh token", ""},
	} {
		t.Run(candidate.name, func(t *testing.T) {
			verifyBrowserIdentityRefreshAcrossInstances(t, candidate.refreshToken)
		})
	}
}

func verifyBrowserIdentityRefreshAcrossInstances(t *testing.T, responseRefreshToken string) {
	const origin = "https://studio.example"
	var refreshCalls atomic.Int32
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
			return
		}
		if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "refresh-1" {
			t.Errorf("unexpected distributed refresh: %v", r.Form)
		}
		if refreshCalls.Add(1) != 1 {
			http.Error(w, "refresh token replay", http.StatusBadRequest)
			return
		}
		entered <- struct{}{}
		<-release
		w.Header().Set("Content-Type", "application/json")
		response := map[string]any{"access_token": "access-2", "id_token": "id-2", "token_type": "Bearer", "expires_in": 3600}
		if responseRefreshToken != "" {
			response["refresh_token"] = responseRefreshToken
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer provider.Close()
	dsn := filepath.Join(t.TempDir(), "shared-studio.db")
	stores := make([]*SQLStore, 2)
	sessions := make([]*Service, 2)
	for index := range stores {
		db, err := sql.Open("sqlite", dsn)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Close() })
		db.SetMaxOpenConns(1)
		if index == 0 {
			migration, err := migrate.New()
			if err != nil {
				t.Fatal(err)
			}
			if err := migration.Up(context.Background(), db); err != nil {
				t.Fatal(err)
			}
		}
		stores[index], err = NewSQLStore(db, []byte("0123456789abcdef0123456789abcdef"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = stores[index].Close(context.Background()) })
		sessions[index], err = New(Config{Issuer: "https://identity.example", Audience: "studio-web", Secure: true, Store: stores[index]}, identityVerifier{})
		if err != nil {
			t.Fatal(err)
		}
	}
	id, _, _, err := sessions[0].ExchangeIDToken(context.Background(), "id-1", "refresh-1")
	if err != nil {
		t.Fatal(err)
	}
	config := LoginConfig{OAuth: oauth2.Config{ClientID: "studio-web", ClientSecret: "backend-only",
		RedirectURL: origin + callbackPath, Scopes: []string{"openid"},
		Endpoint: oauth2.Endpoint{AuthURL: provider.URL + "/authorize", TokenURL: provider.URL + "/token"}},
		CookieKey: []byte("0123456789abcdef0123456789abcdef"), Secure: true, UseIDToken: true, AllowedOrigin: origin}
	handlers := make([]http.Handler, 2)
	for index := range handlers {
		login, err := NewLogin(config, sessions[index])
		if err != nil {
			t.Fatal(err)
		}
		handlers[index], err = login.DatlyHTTP()
		if err != nil {
			t.Fatal(err)
		}
	}
	call := func(handler http.Handler) <-chan *httptest.ResponseRecorder {
		result := make(chan *httptest.ResponseRecorder, 1)
		go func() {
			request := httptest.NewRequest(http.MethodPost, tokenPath, nil)
			request.Header.Set("Origin", origin)
			request.AddCookie(&http.Cookie{Name: DefaultCookieName, Value: id})
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			result <- recorder
		}()
		return result
	}
	first := call(handlers[0])
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("first instance did not start refresh")
	}
	second := call(handlers[1])
	time.Sleep(100 * time.Millisecond)
	close(release)
	for _, result := range []<-chan *httptest.ResponseRecorder{first, second} {
		select {
		case response := <-result:
			if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"id_token":"id-2"`) {
				t.Fatalf("distributed refresh status=%d body=%s", response.Code, response.Body.String())
			}
		case <-time.After(5 * time.Second):
			t.Fatal("distributed refresh did not finish")
		}
	}
	if refreshCalls.Load() != 1 {
		t.Fatalf("refresh token was used %d times", refreshCalls.Load())
	}
	stored, found, err := sessions[0].store.Get(context.Background(), id)
	if err != nil || !found || stored.token != "id-2" {
		t.Fatalf("refreshed session was not stored: found=%v err=%v", found, err)
	}
	wantRefreshToken := responseRefreshToken
	if wantRefreshToken == "" {
		wantRefreshToken = "refresh-1"
	}
	if stored.refreshToken != wantRefreshToken {
		t.Fatalf("stored refresh token=%q, want %q", stored.refreshToken, wantRefreshToken)
	}
}

func TestBrowserIdentityRejectsRefreshedIDTokenBeyondSixtyMinutes(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "access-2", "refresh_token": "refresh-2",
			"id_token": "id-long", "token_type": "Bearer", "expires_in": 3600})
	}))
	defer provider.Close()
	sessions, err := New(Config{Issuer: "https://identity.example", Audience: "studio-web"}, identityVerifier{})
	if err != nil {
		t.Fatal(err)
	}
	id, _, _, err := sessions.ExchangeIDToken(context.Background(), "id-1", "refresh-1")
	if err != nil {
		t.Fatal(err)
	}
	login, err := NewLogin(LoginConfig{OAuth: oauth2.Config{ClientID: "studio-web", RedirectURL: "https://studio.example" + callbackPath,
		Scopes: []string{"openid"}, Endpoint: oauth2.Endpoint{AuthURL: provider.URL + "/authorize", TokenURL: provider.URL + "/token"}},
		CookieKey: []byte("0123456789abcdef0123456789abcdef"), UseIDToken: true, AllowedOrigin: "https://studio.example"}, sessions)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := login.DatlyHTTP()
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, tokenPath, nil)
	request.Header.Set("Origin", "https://studio.example")
	request.AddCookie(&http.Cookie{Name: DefaultCookieName, Value: id})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("overlong refresh status=%d body=%s", response.Code, response.Body.String())
	}
	stored, found, err := sessions.store.Get(context.Background(), id)
	if err != nil || !found || stored.token != "id-1" || stored.refreshToken != "refresh-1" {
		t.Fatalf("rejected refresh changed session: found=%v err=%v", found, err)
	}
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
