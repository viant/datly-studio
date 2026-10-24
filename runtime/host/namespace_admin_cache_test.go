package host

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/viant/datly-studio/runtime/namespacemcp"
	dexec "github.com/viant/datly/exec"
)

type adminCacheSource struct{ calls int }

func (*adminCacheSource) ServeHTTP(http.ResponseWriter, *http.Request) {}
func (s *adminCacheSource) InvalidatePublishedCacheView(_ context.Context, report string, version int, path, view string) ([]dexec.CacheInvalidation, error) {
	s.calls++
	if report != "reader" || version != 1 || path != "/reader" || view != "rows" {
		return nil, context.Canceled
	}
	return []dexec.CacheInvalidation{{View: view, Scope: "all", Generation: "generation-2"}}, nil
}

func TestNamespaceCacheAdminUsesExistingDeploymentToken(t *testing.T) {
	source := &adminCacheSource{}
	manager, err := namespacemcp.NewHTTP(func(context.Context, namespacemcp.Endpoint) (http.Handler, error) { return source, nil })
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(context.Background())
	id := strings.Repeat("a", 64)
	if _, err = manager.Start(context.Background(), id, 0); err != nil {
		t.Fatal(err)
	}
	handler := (&NamespaceGroup{Listeners: manager}).AdminHandler("deployment-token")
	body := []byte(`{"namespaceId":"` + id + `","reportId":"reader","versionNo":1,"routePath":"/reader","view":"rows"}`)
	for _, check := range []struct {
		token, id string
		want      int
	}{
		{"", id, 401},
		{"wrong", id, 401},
		{"deployment-token", strings.Repeat("b", 64), 409},
		{"deployment-token", id, 200},
	} {
		payload := bytes.Replace(body, []byte(id), []byte(check.id), 1)
		request := httptest.NewRequest(http.MethodPost, "/_studio/cache/invalidate", bytes.NewReader(payload))
		if check.token != "" {
			request.Header.Set("X-Studio-Runtime-Token", check.token)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != check.want {
			t.Fatalf("token=%q id=%s status=%d want=%d", check.token, check.id[:8], response.Code, check.want)
		}
	}
	if source.calls != 1 {
		t.Fatalf("unauthorized or foreign request reached source: calls=%d", source.calls)
	}
}
