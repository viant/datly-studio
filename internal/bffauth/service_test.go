package bffauth

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/store/sql/migrate"
	"github.com/viant/scy/auth/jwt"
	_ "modernc.org/sqlite"
)

type verifierStub struct{}

func (verifierStub) VerifyClaims(_ context.Context, token string) (*jwt.Claims, error) {
	claims := &jwt.Claims{}
	claims.Subject = token
	return claims, nil
}

type expiryVerifier struct{ expires time.Time }

func (v expiryVerifier) VerifyClaims(_ context.Context, token string) (*jwt.Claims, error) {
	claims := &jwt.Claims{}
	claims.Subject = token
	claims.ExpiresAt = jwtv5.NewNumericDate(v.expires)
	return claims, nil
}

type claimsVerifier struct{ claims *jwt.Claims }

func (v claimsVerifier) VerifyClaims(context.Context, string) (*jwt.Claims, error) {
	return v.claims, nil
}

func TestProxyExpandsSessionToBearerAndStripsControlHeaders(t *testing.T) {
	var authorization, cookie, path, adminToken, developmentSubject, requestID string
	var calls int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		authorization, cookie, path = r.Header.Get("Authorization"), r.Header.Get("Cookie"), r.URL.Path
		adminToken, developmentSubject, requestID = r.Header.Get("X-Studio-Runtime-Token"), r.Header.Get("X-Studio-Development-Subject"), r.Header.Get("X-Request-ID")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()
	target, _ := url.Parse(upstream.URL)
	service, _ := New(Config{}, verifierStub{})
	id, _, _, err := service.Exchange(context.Background(), "jwt-token")
	if err != nil {
		t.Fatal(err)
	}
	proxy, err := service.Proxy(target, "/v1/studio/runtime")
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/studio/runtime/readers/vendor", nil)
	request.AddCookie(&http.Cookie{Name: DefaultCookieName, Value: id})
	request.Header.Set("X-Studio-Runtime-Token", "attacker-token")
	request.Header.Set("X-Studio-Development-Subject", "attacker")
	request.Header.Set("X-Request-ID", "request-1")
	response := httptest.NewRecorder()
	proxy.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || authorization != "Bearer jwt-token" || cookie != "" || path != "/readers/vendor" || adminToken != "" || developmentSubject != "" || requestID != "request-1" {
		t.Fatalf("status=%d auth=%q cookie=%q path=%q admin=%q development=%q requestID=%q", response.Code, authorization, cookie, path, adminToken, developmentSubject, requestID)
	}
	admin := httptest.NewRequest(http.MethodPost, "/v1/studio/runtime/_studio/reload", nil)
	admin.AddCookie(&http.Cookie{Name: DefaultCookieName, Value: id})
	adminResponse := httptest.NewRecorder()
	proxy.ServeHTTP(adminResponse, admin)
	if adminResponse.Code != http.StatusNotFound || calls != 1 {
		t.Fatalf("admin status=%d upstream calls=%d", adminResponse.Code, calls)
	}
	preservingProxy, err := service.Proxy(target, "/")
	if err != nil {
		t.Fatal(err)
	}
	aclRequest := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/acl.list", strings.NewReader(`{"reportId":"r1"}`))
	aclRequest.AddCookie(&http.Cookie{Name: DefaultCookieName, Value: id})
	aclRequest.Header.Set("X-Studio-Development-Subject", "attacker")
	aclResponse := httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.Handle("/v1/studio/sdk/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) }))
	mux.Handle("/v1/studio/sdk/acl.list", preservingProxy)
	mux.Handle("/v1/studio/sdk/connectors.get", preservingProxy)
	mux.Handle("/v1/studio/sdk/connectors.list", preservingProxy)
	mux.Handle("/v1/studio/sdk/reports.get", preservingProxy)
	mux.Handle("/v1/studio/sdk/reports.list", preservingProxy)
	mux.ServeHTTP(aclResponse, aclRequest)
	if aclResponse.Code != http.StatusNoContent || path != "/v1/studio/sdk/acl.list" ||
		authorization != "Bearer jwt-token" || cookie != "" || developmentSubject != "" || calls != 2 {
		t.Fatalf("native ACL proxy status=%d path=%q auth=%q cookie=%q dev=%q calls=%d",
			aclResponse.Code, path, authorization, cookie, developmentSubject, calls)
	}
	reportRequest := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/reports.list", strings.NewReader(`{"limit":1}`))
	reportRequest.AddCookie(&http.Cookie{Name: DefaultCookieName, Value: id})
	reportResponse := httptest.NewRecorder()
	mux.ServeHTTP(reportResponse, reportRequest)
	if reportResponse.Code != http.StatusNoContent || path != "/v1/studio/sdk/reports.list" || calls != 3 {
		t.Fatalf("native report proxy status=%d path=%q calls=%d", reportResponse.Code, path, calls)
	}
	getRequest := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/reports.get", strings.NewReader(`{"id":"r1"}`))
	getRequest.AddCookie(&http.Cookie{Name: DefaultCookieName, Value: id})
	getResponse := httptest.NewRecorder()
	mux.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusNoContent || path != "/v1/studio/sdk/reports.get" || calls != 4 {
		t.Fatalf("native report-get proxy status=%d path=%q calls=%d", getResponse.Code, path, calls)
	}
	connectorRequest := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/connectors.list", strings.NewReader(`{"limit":1}`))
	connectorRequest.AddCookie(&http.Cookie{Name: DefaultCookieName, Value: id})
	connectorResponse := httptest.NewRecorder()
	mux.ServeHTTP(connectorResponse, connectorRequest)
	if connectorResponse.Code != http.StatusNoContent || path != "/v1/studio/sdk/connectors.list" || calls != 5 {
		t.Fatalf("native connector proxy status=%d path=%q calls=%d", connectorResponse.Code, path, calls)
	}
	connectorGet := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/connectors.get", strings.NewReader(`{"name":"main"}`))
	connectorGet.AddCookie(&http.Cookie{Name: DefaultCookieName, Value: id})
	connectorGetResponse := httptest.NewRecorder()
	mux.ServeHTTP(connectorGetResponse, connectorGet)
	if connectorGetResponse.Code != http.StatusNoContent || path != "/v1/studio/sdk/connectors.get" || calls != 6 {
		t.Fatalf("native connector-get proxy status=%d path=%q calls=%d", connectorGetResponse.Code, path, calls)
	}
}

func TestBFFSessionExchangeAuthenticateAndLogout(t *testing.T) {
	service, err := New(Config{CookieName: "studio_session"}, verifierStub{})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	service.Register(mux)
	exchange := httptest.NewRequest(http.MethodPost, "/v1/studio/auth/session", nil)
	exchange.Header.Set("Authorization", "Bearer owner-a")
	exchangeResponse := httptest.NewRecorder()
	mux.ServeHTTP(exchangeResponse, exchange)
	if exchangeResponse.Code != http.StatusCreated || len(exchangeResponse.Result().Cookies()) != 1 {
		t.Fatalf("exchange=%d cookies=%v", exchangeResponse.Code, exchangeResponse.Result().Cookies())
	}
	cookie := exchangeResponse.Result().Cookies()[0]
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie=%+v", cookie)
	}

	request := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/reports.list", nil)
	request.AddCookie(cookie)
	ctx, err := service.Authenticate(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	principal, ok := sdk.PrincipalFromContext(ctx)
	if !ok || principal.Subject != "owner-a" {
		t.Fatalf("principal=%+v", principal)
	}

	me := httptest.NewRequest(http.MethodGet, "/v1/studio/auth/me", nil)
	me.AddCookie(cookie)
	meResponse := httptest.NewRecorder()
	mux.ServeHTTP(meResponse, me)
	if meResponse.Code != http.StatusOK {
		t.Fatalf("me=%d %s", meResponse.Code, meResponse.Body.String())
	}

	logout := httptest.NewRequest(http.MethodDelete, "/v1/studio/auth/session", nil)
	logout.AddCookie(cookie)
	logoutResponse := httptest.NewRecorder()
	mux.ServeHTTP(logoutResponse, logout)
	if logoutResponse.Code != http.StatusNoContent {
		t.Fatalf("logout=%d", logoutResponse.Code)
	}
	if _, err = service.Authenticate(context.Background(), request); err == nil {
		t.Fatal("logged-out session authenticated")
	}
}

func TestBFFRejectsMissingBearerAndCookie(t *testing.T) {
	service, _ := New(Config{}, verifierStub{})
	response := httptest.NewRecorder()
	service.handleSession(response, httptest.NewRequest(http.MethodPost, "/v1/studio/auth/session", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", response.Code)
	}
	if _, err := service.Authenticate(context.Background(), httptest.NewRequest(http.MethodPost, "/", nil)); err == nil {
		t.Fatal("missing cookie authenticated")
	}
}

func TestBFFRequiresJWTSubjectNotDisplayClaims(t *testing.T) {
	_, err := principalFromClaims(&jwt.Claims{Username: "display-user", Email: "display@example.com"})
	if err == nil {
		t.Fatal("username or email authenticated without JWT sub")
	}
}

func TestBFFSessionNeverOutlivesVerifiedJWT(t *testing.T) {
	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	tokenExpiry := now.Add(5 * time.Minute)
	service, err := New(Config{TTL: 12 * time.Hour}, expiryVerifier{expires: tokenExpiry})
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return now }
	id, _, expires, err := service.Exchange(context.Background(), "owner")
	if err != nil || !expires.Equal(tokenExpiry) {
		t.Fatalf("expires=%v err=%v", expires, err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{Name: DefaultCookieName, Value: id})
	now = tokenExpiry.Add(time.Second)
	if _, err = service.Authenticate(context.Background(), request); err == nil {
		t.Fatal("session authenticated after JWT expiry")
	}
}

func TestBFFRequiresConfiguredIssuerAndAudience(t *testing.T) {
	claims := &jwt.Claims{RegisteredClaims: jwtv5.RegisteredClaims{
		Subject: "owner", Issuer: "https://issuer.example", Audience: jwtv5.ClaimStrings{"studio", "another"}, ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(time.Hour)),
	}}
	service, err := New(Config{Issuer: "https://issuer.example", Audience: "studio"}, claimsVerifier{claims: claims})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err = service.Exchange(context.Background(), "signed-token"); err != nil {
		t.Fatalf("matching token rejected: %v", err)
	}
	for name, config := range map[string]Config{
		"issuer":   {Issuer: "https://other.example", Audience: "studio"},
		"audience": {Issuer: "https://issuer.example", Audience: "other-api"},
	} {
		t.Run(name, func(t *testing.T) {
			candidate, newErr := New(config, claimsVerifier{claims: claims})
			if newErr != nil {
				t.Fatal(newErr)
			}
			if _, _, _, exchangeErr := candidate.Exchange(context.Background(), "signed-token"); exchangeErr == nil {
				t.Fatal("mismatched token binding was accepted")
			}
		})
	}
	withoutExpiry := *claims
	withoutExpiry.ExpiresAt = nil
	missingExpiry, err := New(Config{Issuer: "https://issuer.example", Audience: "studio"}, claimsVerifier{claims: &withoutExpiry})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := missingExpiry.Exchange(context.Background(), "signed-token"); err == nil {
		t.Fatal("bound access token without expiry created a session")
	}
}

func TestSQLStoreSharesEncryptedSessionsAcrossInstances(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	migration, _ := migrate.New()
	if err = migration.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	key := []byte("0123456789abcdef0123456789abcdef")
	firstStore, err := NewSQLStore(db, key)
	if err != nil {
		t.Fatal(err)
	}
	secondStore, err := NewSQLStore(db, key)
	if err != nil {
		t.Fatal(err)
	}
	first, _ := New(Config{Store: firstStore}, verifierStub{})
	second, _ := New(Config{Store: secondStore}, verifierStub{})
	id, _, _, err := first.Exchange(context.Background(), "owner-secret-token")
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{Name: DefaultCookieName, Value: id})
	ctx, err := second.Authenticate(context.Background(), request)
	if err != nil {
		t.Fatalf("second instance did not load session: %v", err)
	}
	principal, ok := sdk.PrincipalFromContext(ctx)
	if !ok || principal.Subject != "owner-secret-token" {
		t.Fatalf("principal=%+v", principal)
	}
	var hash string
	var ciphertext []byte
	if err = db.QueryRow(`SELECT session_id_hash,payload_ciphertext FROM bff_sessions`).Scan(&hash, &ciphertext); err != nil {
		t.Fatal(err)
	}
	if hash == id || strings.Contains(string(ciphertext), "owner-secret-token") {
		t.Fatal("session identifier or bearer was persisted in plaintext")
	}
	if err = secondStore.Delete(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if _, err = first.Authenticate(context.Background(), request); err == nil {
		t.Fatal("logout/delete was not visible to the first instance")
	}
}
