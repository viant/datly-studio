package host

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/predicatepolicy"
	"github.com/viant/datly-studio/studio/namespaces/change"
	"github.com/viant/datly-studio/studio/settings"
)

func TestReloadRechecksNamespaceInternalPolicy(t *testing.T) {
	dql := strings.Replace(unscopedRecordsDQL, "#define($_ = $Records", "#setting($_ = $mcp('records.list','List records'))\n#define($_ = $Records", 1)
	fixture, err := newScopedHost(t, map[string]string{"records": dql}, map[string]authz.Policy{"records": {Mode: "public"}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	service := fixture.service
	token := fixture.token(t, "alice", []authz.Entity{})
	if status, body := fixture.get(t, "/records", token, nil); status != http.StatusOK || !strings.Contains(body, "alpha-101") {
		t.Fatalf("baseline reader unavailable: %d %s", status, body)
	}
	if _, body := fixture.callTool(t, token, "records.list", nil); !strings.Contains(body, "alpha-101") {
		t.Fatalf("baseline tool unavailable: %s", body)
	}
	if _, err := service.studio.ExecContext(ctx, `INSERT OR IGNORE INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at) VALUES('owner','general','General','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP); INSERT INTO namespace_settings(owner_id,namespace,setting_key,value_json,updated_at,updated_by) VALUES('owner','general','security','{"requireInternalComponents":true}',CURRENT_TIMESTAMP,'fixture')`); err != nil {
		t.Fatal(err)
	}
	applyTestNamespaceSecurity(ctx, t, service, 2, true)
	assertDenied := func() {
		t.Helper()
		if status, body := fixture.get(t, "/records", token, nil); status != http.StatusForbidden || strings.Contains(body, "alpha-101") {
			t.Fatalf("old HTTP generation bypassed tightened policy: %d %s", status, body)
		}
		if _, body := fixture.callTool(t, token, "records.list", nil); strings.Contains(body, "alpha-101") || !strings.Contains(body, `"code":-32600`) || !strings.Contains(body, "authorization denied") {
			t.Fatalf("old MCP generation bypassed tightened policy: %s", body)
		}
	}
	assertDenied()
	var denied *predicatepolicy.InternalComponentRequiredError
	if err := service.Reload(ctx, 0); !errors.As(err, &denied) {
		t.Fatalf("reload accepted public component under internal-only policy: %v", err)
	}
	assertDenied()
	if _, err := service.studio.ExecContext(ctx, `UPDATE namespace_settings SET value_json='{"requireInternalComponents":false}' WHERE owner_id='owner' AND namespace='general' AND setting_key='security'`); err != nil {
		t.Fatal(err)
	}
	applyTestNamespaceSecurity(ctx, t, service, 3, false)
	if err := service.Reload(ctx, 0); err != nil {
		t.Fatalf("explicitly disabled policy still blocked reload: %v", err)
	}
	if status, body := fixture.get(t, "/records", token, nil); status != http.StatusOK || !strings.Contains(body, "alpha-101") {
		t.Fatalf("explicit policy relaxation did not restore serving: %d %s", status, body)
	}
}

func applyTestNamespaceSecurity(ctx context.Context, t *testing.T, service *Service, revision int64, internal bool) {
	t.Helper()
	if err := service.ApplyNamespaceSettings(ctx, change.Identity{NamespaceID: namespaceaccess.ID("owner", "general"), OwnerID: "owner", Name: "general", SettingsChanged: true, SecurityChanged: true, Revision: revision, Security: &settings.Security{RequireInternalComponents: internal}}); err != nil {
		t.Fatal(err)
	}
}
