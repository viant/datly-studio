package namespacemcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type nativeTestOwner struct {
	http.Handler
	closed  atomic.Bool
	entered chan struct{}
	exited  chan struct{}
}

func (o *nativeTestOwner) WrapHTTP(http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "composition", 418) })
}
func (o *nativeTestOwner) WrapNativeHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test" {
			http.Error(w, "unauthorized", 401)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (o *nativeTestOwner) Close(context.Context) error { o.closed.Store(true); return nil }

func TestNativeHandlerReusesActiveRuntimeAndDrains(t *testing.T) {
	for _, shared := range []bool{false, true} {
		name := "dedicated"
		if shared {
			name = "shared"
		}
		t.Run(name, func(t *testing.T) {
			var builds atomic.Int32
			var current *nativeTestOwner
			factory := func(context.Context, Endpoint) (http.Handler, error) {
				builds.Add(1)
				owner := &nativeTestOwner{entered: make(chan struct{}), exited: make(chan struct{})}
				owner.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/wait" {
						close(owner.entered)
						<-r.Context().Done()
						if owner.closed.Load() {
							t.Error("source closed before request drain")
						}
						close(owner.exited)
					}
					w.WriteHeader(204)
				})
				current = owner
				return owner, nil
			}
			var m *Manager
			var err error
			if shared {
				m, err = NewSharedHTTP(factory, "127.0.0.1:0")
			} else {
				m, err = NewHTTP(factory)
			}
			if err != nil {
				t.Fatal(err)
			}
			defer m.Close(context.Background())
			id := strings.Repeat("a", 64)
			if _, err = m.Start(context.Background(), id, 0); err != nil {
				t.Fatal(err)
			}
			h := m.NativeHandler(id)
			request := func(path string, authorized bool) int {
				r := httptest.NewRequest("POST", path, nil)
				if authorized {
					r.Header.Set("Authorization", "Bearer test")
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				return w.Code
			}
			if got := request("/mcp", false); got != 401 {
				t.Fatalf("unguarded native handler: %d", got)
			}
			if got := request("/mcp", true); got != 204 {
				t.Fatalf("native dispatch: %d", got)
			}
			if builds.Load() != 1 {
				t.Fatal("native access rebuilt runtime")
			}
			owner := current
			done := make(chan int, 1)
			go func() { done <- request("/wait", true) }()
			<-owner.entered
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err = m.Stop(ctx, id); err != nil {
				t.Fatal(err)
			}
			<-done
			if !owner.closed.Load() {
				t.Fatal("source not closed after drain")
			}
			if got := request("/mcp", true); got != 503 {
				t.Fatalf("stale runtime accessible: %d", got)
			}
			if _, err = m.Start(context.Background(), id, 0); err != nil {
				t.Fatal(err)
			}
			if got := request("/mcp", true); got != 204 || builds.Load() != 2 {
				t.Fatalf("handler did not follow replacement: %d builds=%d", got, builds.Load())
			}
		})
	}
}
