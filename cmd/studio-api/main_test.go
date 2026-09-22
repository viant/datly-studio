package main

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/viant/datly-studio/sdk/httptransport"
)

func TestAuthenticatedDeploymentRequiresExplicitOriginAndRuntimeToken(t *testing.T) {
	if err := validateGatewayMode("typo"); err == nil {
		t.Fatal("unsupported gateway mode was accepted")
	}
	if value, err := resolveRuntimeAdminToken(string(httptransport.Development), ""); err != nil || value != localRuntimeAdminToken {
		t.Fatalf("development token=%q err=%v", value, err)
	}
	for _, value := range []string{"", localRuntimeAdminToken} {
		if _, err := resolveRuntimeAdminToken(string(httptransport.Authenticated), value); err == nil {
			t.Fatalf("authenticated token %q was accepted", value)
		}
	}
	if value, err := resolveRuntimeAdminToken(string(httptransport.Authenticated), "production-secret"); err != nil || value != "production-secret" {
		t.Fatalf("authenticated token=%q err=%v", value, err)
	}
	if value, err := resolveAllowedOrigin(string(httptransport.Development), ""); err != nil || value != "http://127.0.0.1:5173" {
		t.Fatalf("development origin=%q err=%v", value, err)
	}
	if _, err := resolveAllowedOrigin(string(httptransport.Authenticated), ""); err == nil {
		t.Fatal("authenticated mode accepted an empty origin")
	}
	if _, err := resolveAllowedOrigin(string(httptransport.Authenticated), "https://studio.example.com/path"); err == nil {
		t.Fatal("origin with path was accepted")
	}
}

func TestAuthenticatedConfigRequiresTokenBindingAndSessionKey(t *testing.T) {
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	issuer, audience, decoded, err := resolveAuthenticatedConfig("https://issuer.example/jwks", "https://issuer.example", "datly-studio", key)
	if err != nil || issuer != "https://issuer.example" || audience != "datly-studio" || len(decoded) != 32 {
		t.Fatalf("issuer=%q audience=%q key=%d err=%v", issuer, audience, len(decoded), err)
	}
	for name, input := range map[string][4]string{
		"cert":     {"", "issuer", "audience", key},
		"issuer":   {"cert", "", "audience", key},
		"audience": {"cert", "issuer", "", key},
		"key":      {"cert", "issuer", "audience", "short"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, _, err := resolveAuthenticatedConfig(input[0], input[1], input[2], input[3]); err == nil {
				t.Fatal("invalid authenticated configuration was accepted")
			}
		})
	}
}

func TestCORSUsesExactOriginAndRejectsCrossOriginRequests(t *testing.T) {
	called := false
	handler := cors("https://studio.example.com", false, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	preflight := httptest.NewRequest(http.MethodOptions, "/v1/studio/sdk/reports.list", nil)
	preflight.Header.Set("Origin", "https://studio.example.com")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, preflight)
	if response.Code != http.StatusNoContent || response.Header().Get("Access-Control-Allow-Origin") != "https://studio.example.com" || response.Header().Get("Access-Control-Expose-Headers") != "X-Request-ID" || !strings.Contains(response.Header().Get("Access-Control-Allow-Headers"), "Mcp-Protocol-Version") || !regexp.MustCompile(`(^|,\s*)Origin($|,)`).MatchString(response.Header().Get("Vary")) {
		t.Fatalf("preflight status=%d headers=%v", response.Code, response.Header())
	}

	blocked := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/reports.list", nil)
	blocked.Header.Set("Origin", "https://evil.example")
	blockedResponse := httptest.NewRecorder()
	handler.ServeHTTP(blockedResponse, blocked)
	if blockedResponse.Code != http.StatusForbidden || called {
		t.Fatalf("blocked status=%d called=%v", blockedResponse.Code, called)
	}

	sameSite := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/reports.list", nil)
	sameSiteResponse := httptest.NewRecorder()
	handler.ServeHTTP(sameSiteResponse, sameSite)
	if !called {
		t.Fatal("originless server request was blocked")
	}

	strict := cors("https://studio.example.com", true, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	strictResponse := httptest.NewRecorder()
	strict.ServeHTTP(strictResponse, httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/reports.list", nil))
	if strictResponse.Code != http.StatusForbidden {
		t.Fatalf("authenticated originless mutation status=%d", strictResponse.Code)
	}
}

func TestRequestIDIsGeneratedAndForwarded(t *testing.T) {
	var forwarded string
	handler := requestIDs(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) { forwarded = request.Header.Get("X-Request-ID") }))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	requestID := response.Header().Get("X-Request-ID")
	if !regexp.MustCompile(`^[a-f0-9]{32}$`).MatchString(requestID) || forwarded != requestID {
		t.Fatalf("request id response=%q forwarded=%q", requestID, forwarded)
	}
}
