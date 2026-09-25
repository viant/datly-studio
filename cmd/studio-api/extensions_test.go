package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/viant/datly-studio/internal/bffauth"
	"github.com/viant/datly-studio/sdk/httptransport"
	"github.com/viant/scy/auth/jwt"
)

type extensionVerifier struct{}

func (extensionVerifier) VerifyClaims(_ context.Context, token string) (*jwt.Claims, error) {
	claims := &jwt.Claims{}
	claims.Subject = token
	return claims, nil
}

func TestExtensionProxyConfiguration(t *testing.T) {
	if config, err := resolveExtensionProxyConfig(string(httptransport.Development), "", ""); err != nil || config != nil {
		t.Fatalf("disabled extension config=%+v err=%v", config, err)
	}
	config, err := resolveExtensionProxyConfig(string(httptransport.Authenticated), "https://backend.example", "/api/reports")
	if err != nil || config == nil || config.target.Host != "backend.example" || config.prefix != "/api/reports" {
		t.Fatalf("valid extension config=%+v err=%v", config, err)
	}
	for name, values := range map[string]struct{ mode, backend, prefix string }{
		"development":        {string(httptransport.Development), "https://backend.example", "/api/reports"},
		"missing backend":    {string(httptransport.Authenticated), "", "/api/reports"},
		"missing prefix":     {string(httptransport.Authenticated), "https://backend.example", ""},
		"non-http backend":   {string(httptransport.Authenticated), "file:///tmp/backend", "/api/reports"},
		"backend path":       {string(httptransport.Authenticated), "https://backend.example/admin", "/api/reports"},
		"backend query":      {string(httptransport.Authenticated), "https://backend.example?next=admin", "/api/reports"},
		"backend fragment":   {string(httptransport.Authenticated), "https://backend.example#admin", "/api/reports"},
		"backend userinfo":   {string(httptransport.Authenticated), "https://user:pass@backend.example", "/api/reports"},
		"relative prefix":    {string(httptransport.Authenticated), "https://backend.example", "api/reports"},
		"root prefix":        {string(httptransport.Authenticated), "https://backend.example", "/"},
		"dot prefix":         {string(httptransport.Authenticated), "https://backend.example", "/api/../admin"},
		"empty segment":      {string(httptransport.Authenticated), "https://backend.example", "/api//reports"},
		"encoded prefix":     {string(httptransport.Authenticated), "https://backend.example", "/api/%2fadmin"},
		"control prefix":     {string(httptransport.Authenticated), "https://backend.example", "/_studio/reload"},
		"trailing separator": {string(httptransport.Authenticated), "https://backend.example", "/api/reports/"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := resolveExtensionProxyConfig(values.mode, values.backend, values.prefix); err == nil {
				t.Fatal("unsafe extension proxy configuration was accepted")
			}
		})
	}
}

func TestExtensionProxyForwardsAuthenticatedAllowlistedRouteOnly(t *testing.T) {
	var calls int
	var path, query, host, bearer, cookie, requestID, accept, contentType, protocol, runtimeToken string
	backend := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		calls++
		path, query, host = request.URL.Path, request.URL.RawQuery, request.Host
		bearer, cookie = request.Header.Get("Authorization"), request.Header.Get("Cookie")
		requestID, accept, contentType = request.Header.Get("X-Request-ID"), request.Header.Get("Accept"), request.Header.Get("Content-Type")
		protocol, runtimeToken = request.Header.Get("Mcp-Protocol-Version"), request.Header.Get("X-Studio-Runtime-Token")
		response.Header().Set("Access-Control-Allow-Origin", "https://backend.example")
		response.WriteHeader(http.StatusAccepted)
	}))
	defer backend.Close()
	config, err := resolveExtensionProxyConfig(string(httptransport.Authenticated), backend.URL, "/api/reports")
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := bffauth.New(bffauth.Config{}, extensionVerifier{})
	if err != nil {
		t.Fatal(err)
	}
	id, _, _, err := sessions.Exchange(context.Background(), "server-token")
	if err != nil {
		t.Fatal(err)
	}
	proxy, err := newExtensionProxy(sessions, config)
	if err != nil {
		t.Fatal(err)
	}
	handler := routeExtensionProxy(http.NotFoundHandler(), proxy)
	request := httptest.NewRequest(http.MethodPost, extensionProxyMount+"/api/reports/v1/run?limit=2", strings.NewReader("{}"))
	request.AddCookie(&http.Cookie{Name: bffauth.DefaultCookieName, Value: id})
	request.Header.Set("Authorization", "Bearer attacker-token")
	request.Header.Set("X-Studio-Runtime-Token", "attacker-admin-token")
	request.Header.Set("X-Request-ID", "request-123")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Mcp-Protocol-Version", "2025-03-26")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted || calls != 1 || path != "/api/reports/v1/run" || query != "limit=2" || host != config.target.Host ||
		bearer != "Bearer server-token" || cookie != "" || requestID != "request-123" || accept != "application/json" ||
		contentType != "application/json" || protocol != "2025-03-26" || runtimeToken != "" ||
		response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("status=%d calls=%d path=%q query=%q host=%q bearer=%q cookie=%q requestID=%q accept=%q contentType=%q protocol=%q runtimeToken=%q headers=%v",
			response.Code, calls, path, query, host, bearer, cookie, requestID, accept, contentType, protocol, runtimeToken, response.Header())
	}
	unauthenticated := httptest.NewRecorder()
	handler.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, extensionProxyMount+"/api/reports", nil))
	if unauthenticated.Code != http.StatusUnauthorized || calls != 1 {
		t.Fatalf("unauthenticated status=%d upstream calls=%d", unauthenticated.Code, calls)
	}
	for _, candidate := range []string{
		extensionProxyMount + "/api/reporting",
		extensionProxyMount + "/api/reports-other",
		extensionProxyMount + "/api/reports/../admin",
		extensionProxyMount + "/api/reports//admin",
		extensionProxyMount + "/api/reports/%2e%2e/admin",
		extensionProxyMount + "/api/reports/%2fadmin",
		extensionProxyMount + "/api/reports/%252e%252e/admin",
		extensionProxyMount + "/api/reports/%5cadmin",
		extensionProxyMount + "/api/reports%2fadmin",
		extensionProxyMount + "-other/api/reports",
	} {
		t.Run(candidate, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, candidate, nil)
			request.AddCookie(&http.Cookie{Name: bffauth.DefaultCookieName, Value: id})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusNotFound || calls != 1 {
				t.Fatalf("status=%d upstream calls=%d", response.Code, calls)
			}
		})
	}
	protected := cors("https://studio.example", true, handler)
	for _, origin := range []string{"", "https://other.example"} {
		request := httptest.NewRequest(http.MethodPost, extensionProxyMount+"/api/reports/v1/run", strings.NewReader("{}"))
		request.AddCookie(&http.Cookie{Name: bffauth.DefaultCookieName, Value: id})
		if origin != "" {
			request.Header.Set("Origin", origin)
		}
		response := httptest.NewRecorder()
		protected.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden || calls != 1 {
			t.Fatalf("unsafe origin %q status=%d upstream calls=%d", origin, response.Code, calls)
		}
	}
	request = httptest.NewRequest(http.MethodPost, extensionProxyMount+"/api/reports/v1/run", strings.NewReader("{}"))
	request.AddCookie(&http.Cookie{Name: bffauth.DefaultCookieName, Value: id})
	request.Header.Set("Origin", "https://studio.example")
	response = httptest.NewRecorder()
	protected.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted || calls != 2 {
		t.Fatalf("allowed origin status=%d upstream calls=%d", response.Code, calls)
	}
}

func TestExtensionProxyForwardsAuthenticatedBinaryGet(t *testing.T) {
	const payload = "%PDF-1.7\nprivate export\n"
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/widgets/exports/one" || r.Header.Get("Authorization") != "Bearer server-token" || r.Header.Get("Cookie") != "" {
			t.Errorf("unexpected binary request: %s %s auth=%q cookie=%q", r.Method, r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("Cookie"))
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `attachment; filename="report.pdf"`)
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(payload))
	}))
	defer backend.Close()
	config, err := resolveExtensionProxyConfig(string(httptransport.Authenticated), backend.URL, "/api/widgets")
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := bffauth.New(bffauth.Config{}, extensionVerifier{})
	if err != nil {
		t.Fatal(err)
	}
	id, _, _, err := sessions.Exchange(context.Background(), "server-token")
	if err != nil {
		t.Fatal(err)
	}
	proxy, err := newExtensionProxy(sessions, config)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, extensionProxyMount+"/api/widgets/exports/one", nil)
	request.AddCookie(&http.Cookie{Name: bffauth.DefaultCookieName, Value: id})
	response := httptest.NewRecorder()
	proxy.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != payload || response.Header().Get("Content-Type") != "application/pdf" || response.Header().Get("Cache-Control") != "no-store" || !strings.HasPrefix(response.Header().Get("Content-Disposition"), "attachment;") {
		t.Fatalf("binary extension response: status=%d headers=%+v body=%q", response.Code, response.Header(), response.Body.String())
	}
}
