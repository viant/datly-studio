package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/viant/datly-studio/schema"
)

func TestResourcePolicyNamespaceBindingUpgradeFromV20ThroughV21ToV22(t *testing.T) {
	testPolicyNamespaceBindingUpgrade(t, 20)
}

func TestHistoricalV19ToV20ToV21ToV22PreservesPolicyNamespaceBindings(t *testing.T) {
	testPolicyNamespaceBindingUpgrade(t, 19)
}

func testPolicyNamespaceBindingUpgrade(t *testing.T, startingVersion int) {
	t.Helper()
	ctx := context.Background()
	db := openTestDB(t)
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE resource_policies ADD COLUMN namespace_id VARCHAR(64) NOT NULL DEFAULT ''`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE resource_policy_revisions ADD COLUMN namespace_id VARCHAR(64) NOT NULL DEFAULT ''`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO resource_policy_revisions
		(tenant_id,resource_kind,resource_id,resource_version,revision,policies_json,actor_id,occurred_at,namespace_id,created_at,created_by,updated_at,updated_by)
		VALUES
		('tenant-a','component','report-a','1',1,'[]','creator','2026-09-01 10:00:00','namespace-a','2026-09-01 10:00:00','creator','2026-09-01 10:00:00','creator'),
		('tenant-a','component','report-a','1',2,'[]','editor','2026-09-02 11:00:00','namespace-a','2026-09-02 11:00:00','editor','2026-09-02 11:00:00','editor'),
		('tenant-b','report','report-b','working',3,'[]','other','2026-09-03 12:00:00','namespace-b','2026-09-03 12:00:00','other','2026-09-03 12:00:00','other')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO resource_policies
		(tenant_id,resource_kind,resource_id,resource_version,revision,namespace_id,created_at,created_by,updated_at,updated_by)
		VALUES ('tenant-a','component','report-a','1',2,'namespace-a','2026-09-01 10:00:00','creator','2026-09-02 11:00:00','editor'),
		('tenant-b','report','report-b','working',3,'','2026-09-03 12:00:00','other','2026-09-03 12:00:00','other')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `DROP TABLE resource_policy_namespace_bindings`); err != nil {
		t.Fatal(err)
	}
	if err := schema.SetSQLiteVersion(ctx, db, startingVersion); err != nil {
		t.Fatal(err)
	}
	service, _ := New()
	if err := service.Up(ctx, db); err != nil {
		t.Fatalf("schema upgrade from %d: %v", startingVersion, err)
	}
	if err := assertPolicyNamespaceBinding(ctx, db, "tenant-a", "component", "report-a", "1", 0, "namespace-a", "2026-09-01 10:00:00", "creator", "2026-09-02 11:00:00", "editor"); err != nil {
		t.Fatal(err)
	}
	if err := assertPolicyNamespaceBinding(ctx, db, "tenant-a", "component", "report-a", "1", 1, "namespace-a", "2026-09-01 10:00:00", "creator", "2026-09-01 10:00:00", "creator"); err != nil {
		t.Fatal(err)
	}
	if err := assertPolicyNamespaceBinding(ctx, db, "tenant-a", "component", "report-a", "1", 2, "namespace-a", "2026-09-02 11:00:00", "editor", "2026-09-02 11:00:00", "editor"); err != nil {
		t.Fatal(err)
	}
	var emptyNamespaceBinding int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM resource_policy_namespace_bindings WHERE tenant_id='tenant-b' AND resource_id='report-b' AND policy_revision=0`).Scan(&emptyNamespaceBinding); err != nil || emptyNamespaceBinding != 0 {
		t.Fatalf("empty legacy head namespace created binding rows=%d err=%v", emptyNamespaceBinding, err)
	}
	if err := assertPolicyNamespaceBinding(ctx, db, "tenant-b", "report", "report-b", "working", 3, "namespace-b", "2026-09-03 12:00:00", "other", "2026-09-03 12:00:00", "other"); err != nil {
		t.Fatal(err)
	}
	var bindingCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM resource_policy_namespace_bindings`).Scan(&bindingCount); err != nil || bindingCount != 4 {
		t.Fatalf("namespace binding rows=%d err=%v", bindingCount, err)
	}
	for _, table := range []string{"resource_policies", "resource_policy_revisions"} {
		if err := assertColumnExists(ctx, db, table, "namespace_id"); err != nil {
			t.Fatalf("legacy %s.namespace_id was not preserved: %v", table, err)
		}
	}
	var namespace string
	if err := db.QueryRowContext(ctx, `SELECT namespace_id FROM resource_policies WHERE resource_id='report-a'`).Scan(&namespace); err != nil || namespace != "namespace-a" {
		t.Fatalf("legacy head namespace=%q err=%v", namespace, err)
	}
	if version, err := service.CurrentVersion(ctx, db); err != nil || version != schema.CanonicalVersion {
		t.Fatalf("final schema version=%d err=%v", version, err)
	}
}

func assertPolicyNamespaceBinding(ctx context.Context, db *sql.DB, tenant, kind, id, version string, policyRevision int, namespace, createdAt, createdBy, updatedAt, updatedBy string) error {
	var gotNamespace, gotCreatedAt, gotCreatedBy, gotUpdatedAt, gotUpdatedBy string
	// Verify both namespace and the copied audit actor/timestamps.
	err := db.QueryRowContext(ctx, `SELECT namespace_id,created_at,created_by,updated_at,updated_by FROM resource_policy_namespace_bindings
		WHERE tenant_id=? AND resource_kind=? AND resource_id=? AND resource_version=? AND policy_revision=?`, tenant, kind, id, version, policyRevision).
		Scan(&gotNamespace, &gotCreatedAt, &gotCreatedBy, &gotUpdatedAt, &gotUpdatedBy)
	if err != nil {
		return err
	}
	if gotNamespace != namespace || !sameAuditInstant(gotCreatedAt, createdAt) || gotCreatedBy != createdBy || !sameAuditInstant(gotUpdatedAt, updatedAt) || gotUpdatedBy != updatedBy {
		return fmt.Errorf("binding %s/%s/%s/%s revision %d has namespace/audit %q/%q/%q/%q/%q", tenant, kind, id, version, policyRevision, gotNamespace, gotCreatedAt, gotCreatedBy, gotUpdatedAt, gotUpdatedBy)
	}
	return nil
}

func assertColumnExists(ctx context.Context, db *sql.DB, table, column string) error {
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name=?`, table, column).Scan(&count); err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("column %s.%s is missing", table, column)
	}
	return nil
}
