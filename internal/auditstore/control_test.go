package auditstore

import (
	"context"
	"database/sql"
	"github.com/viant/datly-studio/schema"
	write "github.com/viant/datly-studio/studio/tool_call_audit/control_write"
	_ "modernc.org/sqlite"
	"path/filepath"
	"testing"
	"time"
)

func TestNativeAuditControlsRevisionAndRollback(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, `INSERT INTO namespaces(namespace_id,owner_id,name,title,status,created_at,updated_at) VALUES('ns','owner','alpha','Alpha','active',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	store, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close(ctx)
	original, err := store.Control(ctx, "ns", "", "")
	if err != nil {
		t.Fatal(err)
	}
	row := func(etag int64) *write.AuditScope {
		r := &write.AuditScope{}
		r.SetOwnerId(original.OwnerId)
		r.SetName(original.Name)
		r.SetNamespaceId(original.NamespaceId)
		r.SetEtag(etag)
		return r
	}
	first := row(original.Etag)
	first.SetAuditEnabled(false)
	first.SetUpdatedAt(time.Now().UTC())
	if err = store.Configure(ctx, first); err != nil {
		t.Fatal(err)
	}
	saved, err := store.Control(ctx, "ns", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if saved.AuditEnabled || saved.Etag != original.Etag+1 {
		t.Fatalf("namespace setting/token: %+v", saved)
	}
	stale := row(original.Etag)
	stale.SetAuditEnabled(true)
	stale.SetUpdatedAt(time.Now().UTC())
	if err = store.Configure(ctx, stale); err == nil {
		t.Fatal("stale control revision accepted")
	}
	invalid := row(saved.Etag)
	invalid.SetAuditEnabled(true)
	invalid.SetUpdatedAt(time.Now().UTC())
	setting := &write.ToolSetting{}
	setting.SetNamespaceId(original.NamespaceId)
	empty := ""
	setting.SetToolName(&empty)
	setting.SetEnabled(false)
	setting.SetUpdatedAt(time.Now().UTC())
	setting.SetUpdatedBy(original.OwnerId)
	invalid.SetToolSettings([]*write.ToolSetting{setting})
	if err = store.Configure(ctx, invalid); err == nil {
		t.Fatal("invalid child accepted")
	}
	saved, err = store.Control(ctx, "ns", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if saved.AuditEnabled || saved.Etag != original.Etag+1 {
		t.Fatal("failed child changed parent")
	}
	name := "studio.sdk.example"
	setting.SetToolName(&name)
	insert := row(saved.Etag)
	insert.SetToolSettings([]*write.ToolSetting{setting})
	if err = store.Configure(ctx, insert); err != nil {
		t.Fatal(err)
	}
	var createdAt, createdBy string
	if err = db.QueryRowContext(ctx, "SELECT CAST(created_at AS TEXT),created_by FROM tool_audit_settings WHERE namespace_id='ns' AND tool_name=?", name).Scan(&createdAt, &createdBy); err != nil {
		t.Fatal(err)
	}
	update := row(saved.Etag)
	changed := &write.ToolSetting{}
	changed.SetNamespaceId(original.NamespaceId)
	changed.SetToolName(&name)
	changed.SetEnabled(true)
	changed.SetUpdatedAt(time.Now().UTC())
	actor := "manager"
	changed.SetUpdatedBy(&actor)
	update.SetToolSettings([]*write.ToolSetting{changed})
	if err = store.Configure(ctx, update); err != nil {
		t.Fatal(err)
	}
	var count int
	var enabled bool
	var afterAt, afterBy, updatedBy string
	if err = db.QueryRowContext(ctx, "SELECT COUNT(*),enabled,CAST(created_at AS TEXT),created_by,updated_by FROM tool_audit_settings WHERE namespace_id='ns' AND tool_name=?", name).Scan(&count, &enabled, &afterAt, &afterBy, &updatedBy); err != nil {
		t.Fatal(err)
	}
	if count != 1 || !enabled || afterAt != createdAt || afterBy != createdBy || updatedBy != actor {
		t.Fatal("tool upsert changed creation identity or lost update")
	}
	absent := row(saved.Etag)
	missing := "missing"
	absent.SetName(&missing)
	absent.SetAuditEnabled(false)
	absent.SetUpdatedAt(time.Now().UTC())
	if err = store.Configure(ctx, absent); err == nil {
		t.Fatal("configuration created a missing namespace")
	}
}
