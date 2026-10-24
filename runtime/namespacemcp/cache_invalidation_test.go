package namespacemcp

import (
	"context"
	"net/http"
	"strings"
	"testing"

	dexec "github.com/viant/datly/exec"
)

type cacheControlSource struct {
	namespaceID string
	calls       int
}

func (*cacheControlSource) ServeHTTP(http.ResponseWriter, *http.Request) {}
func (s *cacheControlSource) InvalidatePublishedCacheView(_ context.Context, report string, version int, route, view string) ([]dexec.CacheInvalidation, error) {
	s.calls++
	if report != "reader" || version != 3 || route != "/reader" || view != "rows" {
		return nil, context.Canceled
	}
	return []dexec.CacheInvalidation{{View: view, Scope: "all", Generation: s.namespaceID}}, nil
}

func TestCacheControlUsesOnlySelectedCurrentNamespaceSource(t *testing.T) {
	sources := map[string]*cacheControlSource{}
	manager, err := NewHTTP(func(_ context.Context, endpoint Endpoint) (http.Handler, error) {
		source := &cacheControlSource{namespaceID: endpoint.NamespaceID}
		sources[endpoint.NamespaceID] = source
		return source, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(context.Background())
	a, b := strings.Repeat("a", 64), strings.Repeat("b", 64)
	if _, err = manager.Start(context.Background(), a, 0); err != nil {
		t.Fatal(err)
	}
	if _, err = manager.Start(context.Background(), b, 0); err != nil {
		t.Fatal(err)
	}
	if _, err = manager.InvalidatePublishedCacheView(context.Background(), "other", "reader", 3, "/reader", "rows"); err == nil {
		t.Fatal("foreign namespace admitted")
	}
	result, err := manager.InvalidatePublishedCacheView(context.Background(), a, "reader", 3, "/reader", "rows")
	if err != nil || len(result) != 1 || result[0].Generation != a || sources[a].calls != 1 || sources[b].calls != 0 {
		t.Fatalf("selected source result=%+v err=%v a=%d b=%d", result, err, sources[a].calls, sources[b].calls)
	}
}
