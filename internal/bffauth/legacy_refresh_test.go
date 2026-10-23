package bffauth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/viant/datly-studio/store/sql/migrate"
	"golang.org/x/oauth2"
)

func TestBFFCallbackPreservesRefreshAcrossRestartAndProxyRenews(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "sessions.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	migration, err := migrate.New()
	if err != nil {
		t.Fatal(err)
	}
	if err = migration.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	key := bytes.Repeat([]byte{7}, 32)
	now := time.Now()
	refreshCalls, dispatched := 0, 0
	failRefresh, rejectIdentity := false, false
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Form.Get("grant_type") == "authorization_code" {
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "opaque-access", "id_token": "id-1", "refresh_token": "refresh-1", "token_type": "Bearer"})
			return
		}
		refreshCalls++
		if failRefresh {
			http.Error(w, "temporary provider outage", 503)
			return
		}
		if r.Form.Get("refresh_token") != "refresh-1" {
			t.Error("unexpected refresh credential")
		}
		token := "id-2"
		if rejectIdentity {
			token = "invalid"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "new-opaque-access", "id_token": token, "refresh_token": "refresh-2", "token_type": "Bearer"})
	}))
	defer provider.Close()
	newInstance := func() (*Service, *Login) {
		store, err := NewSQLStore(db, key)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = store.Close(context.Background()) })
		sessions, err := New(Config{Store: store, Issuer: "https://identity.example", Audience: "studio-web"}, identityVerifier{})
		if err != nil {
			t.Fatal(err)
		}
		sessions.now = func() time.Time { return now }
		login, err := NewLogin(LoginConfig{CookieKey: key, OAuth: oauth2.Config{ClientID: "studio-web", RedirectURL: "http://127.0.0.1" + callbackPath, Scopes: []string{"openid"}, Endpoint: oauth2.Endpoint{AuthURL: provider.URL + "/authorize", TokenURL: provider.URL + "/token", AuthStyle: oauth2.AuthStyleInParams}}}, sessions)
		if err != nil {
			t.Fatal(err)
		}
		login.now = func() time.Time { return now }
		return sessions, login
	}
	sessions, login := newInstance()
	pending, err := login.seal(loginState{State: "approved-state", Verifier: "pkce-verifier", Expires: now.Add(time.Minute).Unix(), ReturnTo: "/"})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, callbackPath+"?code=approved&state=approved-state", nil)
	request.AddCookie(&http.Cookie{Name: login.stateCookieName, Value: pending})
	response := httptest.NewRecorder()
	login.handleCallback(response, request)
	if response.Code != http.StatusSeeOther {
		t.Fatalf("callback status=%d", response.Code)
	}
	var cookie *http.Cookie
	for _, c := range response.Result().Cookies() {
		if c.Name == sessions.config.CookieName {
			cookie = c
		}
	}
	if cookie == nil {
		t.Fatal("missing opaque session cookie")
	}
	stored, found, err := sessions.store.Get(ctx, cookie.Value)
	if err != nil || !found || stored.refreshToken != "refresh-1" || !stored.expiresAt.After(now.Add(time.Hour)) {
		t.Fatal("BFF callback discarded refresh or bounded the session to ID-token expiry")
	}
	var ciphertext []byte
	if err = db.QueryRow("SELECT payload_ciphertext FROM bff_sessions WHERE session_id_hash=?", sessionHash(cookie.Value)).Scan(&ciphertext); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ciphertext, []byte("refresh-1")) || bytes.Contains(ciphertext, []byte("id-1")) {
		t.Fatal("token material persisted in plaintext")
	}
	// A new service instance loads the encrypted session after ID-token expiry.
	sessions, _ = newInstance()
	now = now.Add(time.Minute)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dispatched++
		if r.Header.Get("Authorization") != "Bearer id-2" || r.Header.Get("Cookie") != "" {
			t.Error("proxy did not use refreshed identity or forwarded cookies")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()
	target, _ := url.Parse(upstream.URL)
	proxy, err := sessions.Proxy(target, "/runtime")
	if err != nil {
		t.Fatal(err)
	}
	call := func() int {
		r := httptest.NewRequest(http.MethodGet, "/runtime/rows", nil)
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		proxy.ServeHTTP(w, r)
		if w.Header().Get("Set-Cookie") != "" {
			t.Error("renewal changed the opaque session cookie")
		}
		return w.Code
	}
	failRefresh = true
	if code := call(); code != 503 || dispatched != 0 {
		t.Fatalf("outage status=%d dispatched=%d", code, dispatched)
	}
	stored, found, err = sessions.store.Get(ctx, cookie.Value)
	if err != nil || !found || stored.refreshToken != "refresh-1" {
		t.Fatal("outage destroyed the renewable session")
	}
	failRefresh = false
	rejectIdentity = true
	if code := call(); code != 401 || dispatched != 0 {
		t.Fatalf("unverified identity status=%d dispatched=%d", code, dispatched)
	}
	rejectIdentity = false
	if code := call(); code != 204 || dispatched != 1 {
		t.Fatalf("recovery status=%d dispatched=%d", code, dispatched)
	}
	stored, found, err = sessions.store.Get(ctx, cookie.Value)
	if err != nil || !found || stored.refreshToken != "refresh-2" {
		t.Fatal("rotated refresh credential was not persisted")
	}
	if refreshCalls != 3 {
		t.Fatalf("refresh calls=%d", refreshCalls)
	}
}
