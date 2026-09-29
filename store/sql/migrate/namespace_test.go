package migrate

import (
	"context"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"testing"
)

func TestNamespaceOwnershipPreservesUncataloguedLegacyNamespace(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	for _, statement := range []string{
		`CREATE TABLE schema_version(version INTEGER NOT NULL)`,
		`INSERT INTO schema_version VALUES(16)`,
		`CREATE TABLE connectors(name TEXT PRIMARY KEY)`,
		`INSERT INTO connectors VALUES('shared')`,
		`CREATE TABLE components(id TEXT PRIMARY KEY, owner_id TEXT NOT NULL, namespace TEXT NOT NULL)`,
		`INSERT INTO components VALUES('forecast','alice','forecasting'),('other','alice','other')`,
		`CREATE TABLE report_versions(report_id TEXT NOT NULL, version_no INTEGER NOT NULL)`,
		`INSERT INTO report_versions VALUES('forecast',1)`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := migrateNamespaceOwnership(ctx, db); err != nil {
		t.Fatal(err)
	}
	var connectorCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM connectors WHERE name='shared'`).Scan(&connectorCount); err != nil || connectorCount != 1 {
		t.Fatalf("global connector was changed: count=%d err=%v", connectorCount, err)
	}
	var scopedConnectorColumns int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('connectors') WHERE name='namespace_id'`).Scan(&scopedConnectorColumns); err != nil || scopedConnectorColumns != 0 {
		t.Fatalf("global connector acquired namespace ownership: count=%d err=%v", scopedConnectorColumns, err)
	}
	for _, name := range []string{"forecasting", "other"} {
		var id, visibility, roles string
		if err := db.QueryRowContext(ctx, `SELECT namespace_id,visibility,allowed_roles_json FROM namespaces WHERE owner_id='alice' AND name=?`, name).Scan(&id, &visibility, &roles); err != nil {
			t.Fatal(err)
		}
		if id != namespaceaccess.ID("alice", name) || visibility != "private" || roles != "[]" {
			t.Fatalf("unexpected migrated namespace: %s %s %s", id, visibility, roles)
		}
	}
	var rootID, childID string
	if err := db.QueryRowContext(ctx, `SELECT namespace_id FROM components WHERE id='forecast'`).Scan(&rootID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT namespace_id FROM report_versions WHERE report_id='forecast'`).Scan(&childID); err != nil {
		t.Fatal(err)
	}
	if rootID != namespaceaccess.ID("alice", "forecasting") || childID != rootID {
		t.Fatalf("root/child namespace mismatch: %q %q", rootID, childID)
	}
	if err := migrateNamespaceOwnership(ctx, db); err != nil {
		t.Fatalf("repeat migration: %v", err)
	}
}
