package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/viant/datly-studio/internal/bffauth"
	"github.com/viant/scy/auth/jwt"
)

type nativeProxyVerifier struct{}

func (nativeProxyVerifier) VerifyClaims(_ context.Context, token string) (*jwt.Claims, error) {
	claims := &jwt.Claims{}
	claims.Subject = token
	return claims, nil
}

func TestEveryNativeSDKPathUsesAuthenticatedStaticProxy(t *testing.T) {
	var observedPath, observedBearer, observedCookie, observedDevelopment, observedNamespace string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observedPath, observedBearer = r.URL.Path, r.Header.Get("Authorization")
		observedCookie, observedDevelopment = r.Header.Get("Cookie"), r.Header.Get("X-Studio-Development-Subject")
		observedNamespace = r.Header.Get("X-Studio-Namespace")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()
	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := bffauth.New(bffauth.Config{}, nativeProxyVerifier{})
	if err != nil {
		t.Fatal(err)
	}
	id, _, _, err := sessions.Exchange(context.Background(), "owner")
	if err != nil {
		t.Fatal(err)
	}
	nativeSDK, err := sessions.Proxy(target, "/")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	for _, path := range nativeSDKPaths {
		mux.Handle(path, nativeSDK)
	}
	for _, path := range nativeSDKPaths {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, path, nil)
			request.AddCookie(&http.Cookie{Name: bffauth.DefaultCookieName, Value: id})
			request.Header.Set("Authorization", "Bearer attacker")
			request.Header.Set("X-Studio-Development-Subject", "attacker")
			request.Header.Set("X-Studio-Namespace", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != http.StatusNoContent || observedPath != path || observedBearer != "Bearer owner" || observedCookie != "" || observedDevelopment != "" || observedNamespace != request.Header.Get("X-Studio-Namespace") {
				t.Fatalf("native path=%s status=%d upstream=%s bearer=%q cookie=%q development=%q", path, response.Code, observedPath, observedBearer, observedCookie, observedDevelopment)
			}
		})
	}
	generic := httptest.NewRecorder()
	mux.ServeHTTP(generic, httptest.NewRequest(http.MethodPost, "/v1/studio/sdk/future.operation", nil))
	if generic.Code != http.StatusNotFound {
		t.Fatalf("non-Datly SDK route was exposed: status=%d", generic.Code)
	}
	unauthorized := httptest.NewRecorder()
	mux.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodPost, nativeSDKPaths[0], nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("cookie-less native SDK route status=%d", unauthorized.Code)
	}
}

func TestStaticSDKMCPProxyRequiresSessionAndForwardsToolRouting(t *testing.T) {
	var path, bearer, tool, method, cookie string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, bearer, tool, method, cookie = r.URL.Path, r.Header.Get("Authorization"),
			r.Header.Get("Mcp-Name"), r.Header.Get("Mcp-Method"), r.Header.Get("Cookie")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"structuredContent":{"items":[]}}}`))
	}))
	defer upstream.Close()
	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := bffauth.New(bffauth.Config{}, nativeProxyVerifier{})
	if err != nil {
		t.Fatal(err)
	}
	id, _, _, err := sessions.Exchange(context.Background(), "owner")
	if err != nil {
		t.Fatal(err)
	}
	proxy, err := sessions.Proxy(target, "/v1/studio/sdk-mcp")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/v1/studio/sdk-mcp/", proxy)
	request := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk-mcp/mcp", nil)
	request.AddCookie(&http.Cookie{Name: bffauth.DefaultCookieName, Value: id})
	request.Header.Set("Authorization", "Bearer attacker")
	request.Header.Set("Mcp-Method", "tools/call")
	request.Header.Set("Mcp-Name", "studio.sdk.namespaces.list")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK || path != "/mcp" || bearer != "Bearer owner" ||
		tool != "studio.sdk.namespaces.list" || method != "tools/call" || cookie != "" {
		t.Fatalf("static MCP status=%d path=%q bearer=%q tool=%q method=%q cookie=%q", response.Code, path, bearer, tool, method, cookie)
	}
	unauthorized := httptest.NewRecorder()
	mux.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodPost, "/v1/studio/sdk-mcp/mcp", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("cookie-less static MCP status=%d", unauthorized.Code)
	}
}
