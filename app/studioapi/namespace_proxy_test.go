package studioapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/viant/datly-studio/internal/bffauth"
	"github.com/viant/datly-studio/sdk"
)

type namespaceProxyTransport struct {
	endpoints map[string]string
	calls     int
}

type concurrentNamespaceTransport struct {
	id, endpoint string
	calls        atomic.Int32
}

func (s *concurrentNamespaceTransport) Invoke(ctx context.Context, operation string, _ any, output any) error {
	s.calls.Add(1)
	principal, verified := sdk.PrincipalFromContext(ctx)
	id, selected := sdk.NamespaceSelectionFromContext(ctx)
	if !verified || principal.Subject != "owner" || !selected || id != s.id || operation != sdk.OperationRuntimeStatus {
		return fmt.Errorf("verified namespace selection required")
	}
	*output.(*sdk.RuntimeStatus) = sdk.RuntimeStatus{Host: &sdk.RuntimeHost{NamespaceID: id, MCPURL: s.endpoint, Status: "ready"}}
	return nil
}

func TestNamespaceMCPProxyForwardsConcurrentProducerCalls(t *testing.T) {
	id := strings.Repeat("a", 64)
	entered := make(chan string, 2)
	canceled := make(chan string, 2)
	releaseSecond := make(chan struct{})
	var firstCalls, secondCalls, metadataCalls atomic.Int32
	producer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" || r.Header.Get("Authorization") != "Bearer owner" || r.Header.Get("Cookie") != "" || r.Header.Get("X-Studio-Namespace") != id || r.Header.Get("Mcp-Session-Id") != "shared-session" {
			t.Error("producer received incorrect verified route or credential")
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if r.Method == http.MethodGet {
			metadataCalls.Add(1)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		switch r.Header.Get("X-Request-ID") {
		case "first":
			firstCalls.Add(1)
			entered <- "first"
			<-r.Context().Done()
			canceled <- "first"
		case "second":
			secondCalls.Add(1)
			entered <- "second"
			select {
			case <-releaseSecond:
				w.WriteHeader(http.StatusOK)
			case <-r.Context().Done():
				canceled <- "second"
			}
		default:
			t.Error("unexpected producer request")
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer producer.Close()
	transport := &concurrentNamespaceTransport{id: id, endpoint: producer.URL + "/mcp"}
	sessions, err := bffauth.New(bffauth.Config{}, nativeProxyVerifier{})
	if err != nil {
		t.Fatal(err)
	}
	cookie, _, _, err := sessions.Exchange(context.Background(), "owner")
	if err != nil {
		t.Fatal(err)
	}
	target, _ := url.Parse(producer.URL)
	proxy, err := sessions.ProxyWithResolver(target, "/v1/studio/mcp", namespaceMCPTarget(transport))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(proxy)
	defer server.Close()
	client := &http.Client{Timeout: 5 * time.Second}
	request := func(ctx context.Context, method, requestID string) *http.Request {
		req, e := http.NewRequestWithContext(ctx, method, server.URL+"/v1/studio/mcp/mcp", nil)
		if e != nil {
			t.Fatal(e)
		}
		req.AddCookie(&http.Cookie{Name: bffauth.DefaultCookieName, Value: cookie})
		req.Header.Set("X-Studio-Namespace", id)
		req.Header.Set("Mcp-Session-Id", "shared-session")
		if requestID != "" {
			req.Header.Set("X-Request-ID", requestID)
		}
		return req
	}
	firstCtx, cancelFirst := context.WithCancel(context.Background())
	defer cancelFirst()
	type result struct {
		status int
		err    error
	}
	firstDone, secondDone := make(chan result, 1), make(chan result, 1)
	call := func(ctx context.Context, id string, done chan<- result) {
		response, e := client.Do(request(ctx, http.MethodPost, id))
		if e != nil {
			done <- result{err: e}
			return
		}
		response.Body.Close()
		done <- result{status: response.StatusCode}
	}
	go call(firstCtx, "first", firstDone)
	go call(context.Background(), "second", secondDone)
	seen := map[string]bool{}
	for len(seen) < 2 {
		select {
		case id := <-entered:
			seen[id] = true
		case <-time.After(2 * time.Second):
			t.Fatalf("producer calls serialized before dispatch: %v", seen)
		}
	}
	metadata, err := client.Do(request(context.Background(), http.MethodGet, ""))
	if err != nil {
		t.Fatal(err)
	}
	metadata.Body.Close()
	if metadata.StatusCode != http.StatusNoContent {
		t.Fatalf("metadata waited or failed: %d", metadata.StatusCode)
	}
	cancelFirst()
	select {
	case result := <-firstDone:
		if result.err == nil {
			t.Fatalf("canceled first call returned %+v", result)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled call did not release")
	}
	select {
	case id := <-canceled:
		if id != "first" {
			t.Fatalf("cancel propagated to %s", id)
		}
	case <-time.After(time.Second):
		t.Fatal("upstream first call was not canceled")
	}
	select {
	case result := <-secondDone:
		t.Fatalf("second call canceled with first: %+v", result)
	default:
	}
	close(releaseSecond)
	select {
	case result := <-secondDone:
		if result.err != nil || result.status != http.StatusOK {
			t.Fatalf("independent call failed: %+v", result)
		}
	case <-time.After(time.Second):
		t.Fatal("second producer call did not complete")
	}
	if firstCalls.Load() != 1 || secondCalls.Load() != 1 || metadataCalls.Load() != 1 || transport.calls.Load() != 3 {
		t.Fatalf("unexpected forwarding/retry counts: first=%d second=%d metadata=%d resolver=%d", firstCalls.Load(), secondCalls.Load(), metadataCalls.Load(), transport.calls.Load())
	}
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
			expectedPath := "/mcp"
			if id == b {
				expectedPath = "/namespaces/" + b + "/mcp"
			}
			if r.URL.Path != expectedPath || r.Header.Get("Authorization") != "Bearer owner" || r.Header.Get("Cookie") != "" || r.Header.Get("X-Studio-Namespace") != id || r.Header.Get("X-Studio-Runtime-Token") != "" {
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
	transport := &namespaceProxyTransport{endpoints: map[string]string{a: aServer.URL + "/mcp", b: bServer.URL + "/namespaces/" + b + "/mcp"}}
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

func TestNamespaceMCPProxyRejectsDifferentSharedNamespace(t *testing.T) {
	id := strings.Repeat("a", 64)
	transport := &namespaceProxyTransport{endpoints: map[string]string{id: "http://127.0.0.1:8097/namespaces/" + strings.Repeat("b", 64) + "/mcp"}}
	ctx := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "owner"})
	ctx = sdk.WithNamespaceSelection(ctx, id)
	if _, err := namespaceMCPTarget(transport)(ctx, httptest.NewRequest(http.MethodPost, "/mcp", nil)); err == nil {
		t.Fatal("accepted another namespace endpoint")
	}
}
