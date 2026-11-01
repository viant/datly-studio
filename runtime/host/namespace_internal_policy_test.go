package host

import (
	"context"
	"errors"
	"testing"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/internal/predicatepolicy"
)

func TestReloadRechecksNamespaceInternalPolicy(t *testing.T) {
	fixture, err := newScopedHost(t, map[string]string{"records": unscopedRecordsDQL}, map[string]authz.Policy{"records": {Mode: "public"}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	service := fixture.service
	if _, err := service.studio.ExecContext(ctx, `INSERT OR IGNORE INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at) VALUES('owner','general','General','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP); INSERT INTO namespace_settings(owner_id,namespace,setting_key,value_json,updated_at,updated_by) VALUES('owner','general','security','{"requireInternalComponents":true}',CURRENT_TIMESTAMP,'fixture')`); err != nil {
		t.Fatal(err)
	}
	var denied *predicatepolicy.InternalComponentRequiredError
	if err := service.Reload(ctx, 0); !errors.As(err, &denied) {
		t.Fatalf("reload accepted public component under internal-only policy: %v", err)
	}
	if _, err := service.studio.ExecContext(ctx, `UPDATE namespace_settings SET value_json='{"requireInternalComponents":false}' WHERE owner_id='owner' AND namespace='general' AND setting_key='security'`); err != nil {
		t.Fatal(err)
	}
	if err := service.Reload(ctx, 0); err != nil {
		t.Fatalf("explicitly disabled policy still blocked reload: %v", err)
	}
}
