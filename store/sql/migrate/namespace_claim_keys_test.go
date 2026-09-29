package migrate

import (
	"context"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/schema"
	"testing"
)

func TestNamespaceClaimKeyMigrationPreservesAuditAndAllowsIndependentNames(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	for _, query := range []string{
		`CREATE TABLE schema_version(version INTEGER NOT NULL)`, `INSERT INTO schema_version VALUES(18)`,
		`CREATE TABLE components(id TEXT PRIMARY KEY,namespace_id TEXT NOT NULL)`,
		`CREATE TABLE resource_namespace_claims(namespace_id TEXT NOT NULL DEFAULT '',namespace TEXT PRIMARY KEY,report_id TEXT NOT NULL,created_at TEXT NOT NULL,created_by TEXT NOT NULL,updated_at TEXT NOT NULL,updated_by TEXT NOT NULL)`,
	} {
		if _, err := db.ExecContext(ctx, query); err != nil {
			t.Fatal(err)
		}
	}
	a, b := namespaceaccess.ID("owner", "alpha"), namespaceaccess.ID("owner", "beta")
	if _, err := db.ExecContext(ctx, `INSERT INTO components VALUES('a',?),('b',?)`, a, b); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO resource_namespace_claims VALUES('','owner.docs','a','before','author','after','editor')`); err != nil {
		t.Fatal(err)
	}
	service, _ := New()
	if err := service.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	var id, createdBy, updatedBy string
	if err := db.QueryRowContext(ctx, `SELECT namespace_id,created_by,updated_by FROM resource_namespace_claims WHERE report_id='a'`).Scan(&id, &createdBy, &updatedBy); err != nil || id != a || createdBy != "author" || updatedBy != "editor" {
		t.Fatalf("migration lost ownership/audit: %q/%q/%q %v", id, createdBy, updatedBy, err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO resource_namespace_claims VALUES(?,'owner.docs','b','before','author','after','editor')`, b); err != nil {
		t.Fatalf("independent namespace name rejected: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO resource_namespace_claims VALUES(?,'owner.docs','b','before','author','after','editor')`, a); err == nil {
		t.Fatal("duplicate name within one workspace accepted")
	}
	if err := service.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	version, err := schema.SQLiteVersion(ctx, db)
	if err != nil || version != 19 {
		t.Fatalf("schema version=%d err=%v", version, err)
	}
}

func TestNamespaceClaimKeyMigrationRejectsOrphansAtomically(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	for _, query := range []string{
		`CREATE TABLE schema_version(version INTEGER NOT NULL)`, `INSERT INTO schema_version VALUES(18)`,
		`CREATE TABLE components(id TEXT PRIMARY KEY,namespace_id TEXT NOT NULL)`,
		`CREATE TABLE resource_namespace_claims(namespace_id TEXT NOT NULL DEFAULT '',namespace TEXT PRIMARY KEY,report_id TEXT NOT NULL,created_at TEXT NOT NULL,created_by TEXT NOT NULL,updated_at TEXT NOT NULL,updated_by TEXT NOT NULL)`,
		`INSERT INTO resource_namespace_claims VALUES('','owner.docs','missing','before','author','after','editor')`,
	} {
		if _, err := db.ExecContext(ctx, query); err != nil {
			t.Fatal(err)
		}
	}
	service, _ := New()
	if err := service.Up(ctx, db); err == nil {
		t.Fatal("orphan claim migrated")
	}
	version, err := schema.SQLiteVersion(ctx, db)
	if err != nil || version != 18 {
		t.Fatalf("failed migration changed version: %d %v", version, err)
	}
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM resource_namespace_claims WHERE report_id='missing'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("failed migration changed data: %d %v", count, err)
	}
}
