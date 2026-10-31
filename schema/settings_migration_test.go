package schema

import (
	"context"
	"os"
	"testing"
)

func TestNamespaceSettingsMigrationIsExplicitAndHasNoCascadingDelete(t *testing.T) {
	ctx := context.Background()
	db := openSQLite(t, "namespace_settings_upgrade")
	defer db.Close()
	if err := ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "DROP TABLE namespace_settings; ALTER TABLE component_versions DROP COLUMN resource_check_enabled"); err != nil {
		t.Fatal(err)
	}
	if err := SetSQLiteVersion(ctx, db, 26); err != nil {
		t.Fatal(err)
	}
	if err := Verify(ctx, db, "sqlite"); err == nil {
		t.Fatal("old layout accepted without an explicit upgrade")
	}
	script, err := os.ReadFile("migrations/027-namespace-settings.sqlite.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, string(script)); err != nil {
		t.Fatal(err)
	}
	if err = Verify(ctx, db, "sqlite"); err != nil {
		t.Fatal(err)
	}
	version, err := SQLiteVersion(ctx, db)
	if err != nil || version != 27 {
		t.Fatalf("version=%d err=%v", version, err)
	}
	if _, err = db.ExecContext(ctx, "PRAGMA foreign_keys=ON; INSERT INTO namespaces(owner_id,name,title,status,created_at,updated_at) VALUES('owner','alpha','Alpha','active',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP); INSERT INTO namespace_settings(owner_id,namespace,setting_key,value_json,updated_at,updated_by) VALUES('owner','alpha','security','{}',CURRENT_TIMESTAMP,'owner')"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, "DELETE FROM namespaces WHERE owner_id='owner' AND name='alpha'"); err == nil {
		t.Fatal("namespace deletion cascaded or bypassed the settings foreign key")
	}
}
