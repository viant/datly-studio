package studioapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/viant/datly-studio/internal/bffauth"
	"github.com/viant/datly-studio/sdk"
)

type namespaceProxyTransport struct {
	endpoints map[string]string
	calls     int
}

func (s *namespaceProxyTransport) Invoke(ctx context.Context, operation string, input, output any) error {
	s.calls++
	principal, ok := sdk.PrincipalFromContext(ctx)
	id, selected := sdk.NamespaceSelectionFromContext(ctx)
	if !ok || principal.Subject != "owner" || !selected || operation != sdk.OperationRuntimeStatus {
		return fmt.Errorf("verified selection missing")
	}
	endpoint := s.endpoints[id]
	if endpoint == "" {
		return fmt.Errorf("namespace access denied")
	}
	*output.(*sdk.RuntimeStatus) = sdk.RuntimeStatus{Host: &sdk.RuntimeHost{NamespaceID: id, MCPURL: endpoint, Status: "ready"}}
	return nil
}

func TestNamespaceMCPProxyUsesVerifiedSessionAndServerEndpoint(t *testing.T) {
	a, b := strings.Repeat("a", 64), strings.Repeat("b", 64)
	var aCalls, bCalls, fallbackCalls int
	upstream := func(id string, calls *int) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			*calls++
			if r.URL.Path != "/mcp" || r.Header.Get("Authorization") != "Bearer owner" || r.Header.Get("Cookie") != "" || r.Header.Get("X-Studio-Namespace") != id || r.Header.Get("X-Studio-Runtime-Token") != "" {
				t.Error("namespace proxy forwarded incorrect credentials, path or selection")
			}
			w.WriteHeader(http.StatusNoContent)
		}))
	}
	aServer, bServer := upstream(a, &aCalls), upstream(b, &bCalls)
	defer aServer.Close()
	defer bServer.Close()
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fallbackCalls++; w.WriteHeader(204) }))
	defer fallback.Close()
	transport := &namespaceProxyTransport{endpoints: map[string]string{a: aServer.URL + "/mcp", b: bServer.URL + "/mcp"}}
	sessions, err := bffauth.New(bffauth.Config{}, nativeProxyVerifier{})
	if err != nil {
		t.Fatal(err)
	}
	cookie, _, _, err := sessions.Exchange(context.Background(), "owner")
	if err != nil {
		t.Fatal(err)
	}
	target, _ := url.Parse(fallback.URL)
	proxy, err := sessions.ProxyWithResolver(target, "/v1/studio/mcp", namespaceMCPTarget(transport))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id            string
		authenticated bool
		status        int
	}{
		{a, true, 204}, {b, true, 204}, {strings.Repeat("c", 64), true, 403}, {"invalid", true, 400}, {"", true, 403}, {a, false, 401},
	} {
		request := httptest.NewRequest(http.MethodPost, "/v1/studio/mcp/mcp", nil)
		if tc.id != "" {
			request.Header.Set("X-Studio-Namespace", tc.id)
		}
		request.Header.Set("Authorization", "Bearer attacker")
		request.Header.Set("X-Studio-Runtime-Token", "attacker")
		if tc.authenticated {
			request.AddCookie(&http.Cookie{Name: bffauth.DefaultCookieName, Value: cookie})
		}
		response := httptest.NewRecorder()
		proxy.ServeHTTP(response, request)
		if response.Code != tc.status {
			t.Fatalf("namespace=%s status=%d want=%d", tc.id, response.Code, tc.status)
		}
	}
	if aCalls != 1 || bCalls != 1 || fallbackCalls != 0 || transport.calls != 3 {
		t.Fatalf("calls a=%d b=%d fallback=%d lookup=%d", aCalls, bCalls, fallbackCalls, transport.calls)
	}
	duplicate := httptest.NewRequest(http.MethodPost, "/v1/studio/mcp/mcp", nil)
	duplicate.AddCookie(&http.Cookie{Name: bffauth.DefaultCookieName, Value: cookie})
	duplicate.Header.Add("X-Studio-Namespace", a)
	duplicate.Header.Add("X-Studio-Namespace", b)
	duplicateResponse := httptest.NewRecorder()
	proxy.ServeHTTP(duplicateResponse, duplicate)
	if duplicateResponse.Code != 400 || transport.calls != 3 {
		t.Fatal("duplicate namespace selections reached endpoint resolution")
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/studio/mcp/_studio/reload", nil)
	request.Header.Set("X-Studio-Namespace", a)
	request.AddCookie(&http.Cookie{Name: bffauth.DefaultCookieName, Value: cookie})
	response := httptest.NewRecorder()
	proxy.ServeHTTP(response, request)
	if response.Code != 404 || transport.calls != 3 {
		t.Fatal("exposed namespace admin route")
	}
}
