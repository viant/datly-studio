package host

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/viant/authz"
	"github.com/viant/datly/spec"
)

func TestPublishedCacheInvalidationRequiresTrustedExactView(t *testing.T) {
	fixture, err := newScopedHost(t, map[string]string{"records": unscopedRecordsDQL}, map[string]authz.Policy{"records": {Mode: "public"}})
	if err != nil {
		t.Fatal(err)
	}
	service := fixture.service
	ctx := context.Background()
	dql := strings.Replace(unscopedRecordsDQL, "#define", "#setting($_ = $cache('local'))\n#define", 1)
	if _, err = service.studio.ExecContext(ctx, "UPDATE component_versions SET authored_dql=?,generated_dql=? WHERE report_id='records'", dql, dql); err != nil {
		t.Fatal(err)
	}
	service.config.ReaderCaches = map[string]*spec.CacheSettings{"local": {Name: "local", Enabled: true, Provider: "afs", Location: filepath.Join(t.TempDir(), "cache"), TTL: "1m"}}
	if err = service.Reload(ctx, 0); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/api/cache/invalidate/records", strings.NewReader(`{"view":"records","scope":"all"}`))
	request.Header.Set("X-Studio-Runtime-Token", "any-caller-controlled-token")
	response := httptest.NewRecorder()
	service.manager.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("ordinary request invalidated cache: %d %s", response.Code, response.Body.String())
	}
	if _, err = service.InvalidatePublishedCacheView(ctx, "wrong", 1, "/records", "records"); err == nil {
		t.Fatal("different report was admitted")
	}
	if _, err = service.InvalidatePublishedCacheView(ctx, "records", 2, "/records", "records"); err == nil {
		t.Fatal("different version was admitted")
	}
	if _, err = service.InvalidatePublishedCacheView(ctx, "records", 1, "/records", "missing"); err == nil {
		t.Fatal("unknown view was admitted")
	}
	first, err := service.InvalidatePublishedCacheView(ctx, "records", 1, "/records", "records")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.InvalidatePublishedCacheView(ctx, "records", 1, "/records", "records")
	if err != nil || first[0].Generation == second[0].Generation {
		t.Fatalf("one-view generation did not advance: first=%+v second=%+v err=%v", first, second, err)
	}
	service.config.Admin.Token = "deployment-owned-token"
	service.config.NamespaceID = strings.Repeat("a", 64)
	for _, check := range []struct {
		token, namespace string
		want             int
	}{
		{"", service.config.NamespaceID, 401},
		{"forged", service.config.NamespaceID, 401},
		{"deployment-owned-token", strings.Repeat("b", 64), 400},
		{"deployment-owned-token", service.config.NamespaceID, 200},
	} {
		payload := []byte(`{"namespaceId":"` + check.namespace + `","reportId":"records","versionNo":1,"routePath":"/records","view":"records"}`)
		request := httptest.NewRequest(http.MethodPost, "/_studio/cache/invalidate", bytes.NewReader(payload))
		if check.token != "" {
			request.Header.Set("X-Studio-Runtime-Token", check.token)
		}
		response := httptest.NewRecorder()
		service.cacheInvalidateHTTP(response, request)
		if response.Code != check.want {
			t.Fatalf("admin token=%q status=%d want=%d", check.token, response.Code, check.want)
		}
	}
}
