package namespacemcp

import (
	"bytes"
	"context"
	mcpserver "github.com/viant/datly/mcp/server"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSharedPathsIsolationAndDedicatedOverride(t *testing.T) {
	ctx := context.Background()
	m, err := NewShared(factory, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Close(ctx) })
	aID, bID := strings.Repeat("a", 64), strings.Repeat("b", 64)
	a, err := m.Start(ctx, aID, 0)
	if err != nil {
		t.Fatal(err)
	}
	b, err := m.Start(ctx, bID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if a.Port != b.Port || a.URL() == b.URL() {
		t.Fatalf("shared endpoints: %+v %+v", a, b)
	}
	for _, e := range []Endpoint{a, b} {
		own := "read_" + e.NamespaceID[:8]
		other := "read_aaaaaaaa"
		if e.NamespaceID == aID {
			other = "read_bbbbbbbb"
		}
		body := call(t, e, "tools/list", nil)
		if !bytes.Contains(body, []byte(own)) || bytes.Contains(body, []byte(other)) {
			t.Fatalf("cross-namespace catalog: %s", body)
		}
		body = call(t, e, "tools/call", map[string]any{"name": own, "arguments": map[string]any{}})
		if !bytes.Contains(body, []byte(e.NamespaceID)) {
			t.Fatalf("wrong runtime: %s", body)
		}
		body = call(t, e, "tools/call", map[string]any{"name": other, "arguments": map[string]any{}})
		if !bytes.Contains(body, []byte(`"error"`)) {
			t.Fatalf("foreign tool accepted: %s", body)
		}
	}
	for _, path := range []string{"/mcp", "/namespaces/" + strings.Repeat("c", 64) + "/mcp", "/namespaces/" + aID + "/../" + bID + "/mcp"} {
		response, err := http.Get("http://" + a.Address + path)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != 404 {
			t.Fatalf("unscoped path %s returned %d", path, response.StatusCode)
		}
	}
	// A conflicting override leaves the shared endpoint intact.
	if _, err = m.Rebind(ctx, aID, a.Port); err == nil {
		t.Fatal("shared port conflict accepted")
	}
	retained, _ := m.Get(aID)
	if retained.URL() != a.URL() {
		t.Fatal("failed rebind replaced current endpoint")
	}
	free, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := free.Addr().(*net.TCPAddr).Port
	free.Close()
	dedicated, err := m.Rebind(ctx, aID, port)
	if err != nil {
		t.Fatal(err)
	}
	if dedicated.Port != port || dedicated.Path != "" {
		t.Fatalf("dedicated override=%+v", dedicated)
	}
	_ = call(t, dedicated, "tools/list", nil)
	response, err := http.Get(a.URL())
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 404 {
		t.Fatal("dedicated namespace remained mounted in shared router")
	}
	shared, err := m.Rebind(ctx, aID, 0)
	if err != nil || shared.URL() != a.URL() {
		t.Fatalf("return to shared=%+v %v", shared, err)
	}
	if err = m.Stop(ctx, aID); err != nil {
		t.Fatal(err)
	}
	_ = call(t, b, "tools/list", nil)
}

type sharedGuardSource struct {
	*ownedSource
	id string
}

func (s *sharedGuardSource) WrapHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+s.id {
			http.Error(w, "unauthorized", 401)
			return
		}
		if r.URL.Path == "/probe" || strings.HasPrefix(r.URL.Path, "/.well-known/") {
			_, _ = w.Write([]byte(s.id + ":" + r.URL.Path))
			return
		}
		next.ServeHTTP(w, r)
	})
}
func TestSharedNamespaceWrappersAndMetadataPaths(t *testing.T) {
	var closes atomic.Int32
	m, err := NewShared(func(ctx context.Context, e Endpoint) (mcpserver.Config, error) {
		cfg, err := factory(ctx, e)
		if err == nil {
			cfg.Source = &sharedGuardSource{ownedSource: &ownedSource{service: cfg.Service.(*testService), closes: &closes}, id: e.NamespaceID}
			cfg.Service = nil
		}
		return cfg, err
	}, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close(context.Background())
	a, err := m.Start(context.Background(), strings.Repeat("a", 64), 0)
	if err != nil {
		t.Fatal(err)
	}
	b, err := m.Start(context.Background(), strings.Repeat("b", 64), 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path, token, wantBody string
		code                  int
	}{
		{"/namespaces/" + a.NamespaceID + "/probe", a.NamespaceID, a.NamespaceID + ":/probe", 200},
		{"/namespaces/" + b.NamespaceID + "/probe", a.NamespaceID, "", 401},
		{"/.well-known/oauth-protected-resource/namespaces/" + a.NamespaceID + "/oauth/test/mcp", a.NamespaceID, a.NamespaceID + ":/.well-known/oauth-protected-resource/oauth/test/mcp", 200},
	} {
		request, _ := http.NewRequest("GET", "http://"+a.Address+tc.path, nil)
		request.Header.Set("Authorization", "Bearer "+tc.token)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()
		if response.StatusCode != tc.code || tc.code == 200 && string(body) != tc.wantBody {
			t.Fatalf("route %s: %d %s", tc.path, response.StatusCode, body)
		}
	}
}

type sharedWaitingSource struct {
	*ownedSource
	entered chan struct{}
}

func (s *sharedWaitingSource) WrapHTTP(http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(s.entered); <-r.Context().Done() })
}
func TestSharedStopCancelsRequestsBeforeSourceCleanup(t *testing.T) {
	var closes atomic.Int32
	entered := make(chan struct{})
	m, err := NewShared(func(ctx context.Context, e Endpoint) (mcpserver.Config, error) {
		cfg, err := factory(ctx, e)
		if err == nil {
			cfg.Source = &sharedWaitingSource{ownedSource: &ownedSource{service: cfg.Service.(*testService), closes: &closes}, entered: entered}
			cfg.Service = nil
		}
		return cfg, err
	}, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close(context.Background())
	endpoint, err := m.Start(context.Background(), strings.Repeat("a", 64), 0)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		response, err := http.Get(endpoint.URL())
		if err == nil {
			response.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("request did not enter namespace")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err = m.Stop(ctx, endpoint.NamespaceID); err != nil {
		t.Fatal(err)
	}
	if closes.Load() != 1 {
		t.Fatal("namespace source was not cleaned up")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("namespace request was not canceled")
	}
}
