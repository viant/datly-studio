package schema

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSQLiteSchemasAndFixtures(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	studio := openSQLite(t, "studio")
	defer studio.Close()
	if err := ApplySQLite(ctx, studio, "studio"); err != nil {
		t.Fatal(err)
	}
	if err := ApplySQLite(ctx, studio, "studio_seed"); err != nil {
		t.Fatal(err)
	}
	assertCount(t, studio, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'", 25)
	assertCount(t, studio, "SELECT COUNT(*) FROM connectors WHERE name='reporting' AND status='active'", 1)
	assertCount(t, studio, "SELECT COUNT(*) FROM namespaces WHERE owner_id='unit' AND name='general' AND status='active'", 1)

	reporting := openSQLite(t, "reporting")
	defer reporting.Close()
	if err := ApplySQLite(ctx, reporting, "reporting"); err != nil {
		t.Fatal(err)
	}
	if err := ApplySQLite(ctx, reporting, "reporting_seed"); err != nil {
		t.Fatal(err)
	}
	assertCount(t, reporting, "SELECT COUNT(*) FROM VENDOR", 3)
	assertCount(t, reporting, "SELECT COUNT(*) FROM PRODUCT", 3)
}

func TestDropSQLitePreservesSharedAuthzPolicyTables(t *testing.T) {
	ctx := context.Background()
	db := openSQLite(t, "studio_drop_preserves_policy")
	defer db.Close()
	if err := ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	if err := SetSQLiteVersion(ctx, db, CanonicalVersion); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`INSERT INTO resource_policies(tenant_id,resource_kind,resource_id,resource_version,revision,created_by,updated_by) VALUES('tenant','report','r1','1',1,'owner','owner')`,
		`INSERT INTO resource_policy_revisions(tenant_id,resource_kind,resource_id,resource_version,revision,policies_json,actor_id,occurred_at,created_by,updated_by) VALUES('tenant','report','r1','1',1,'[]','owner','2026-10-08 10:00:00','owner','owner')`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := DropSQLite(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"resource_policies", "resource_policy_revisions"} {
		assertTablePresence(t, ctx, db, table, true)
	}
	for _, table := range []string{"connectors", "resource_policy_namespace_bindings", "sqlx_sequence_reservations", "schema_version"} {
		assertTablePresence(t, ctx, db, table, false)
	}
	assertCount(t, db, `SELECT COUNT(*) FROM resource_policies WHERE tenant_id='tenant' AND resource_id='r1' AND revision=1`, 1)
	assertCount(t, db, `SELECT COUNT(*) FROM resource_policy_revisions WHERE tenant_id='tenant' AND resource_id='r1' AND revision=1`, 1)
}

func TestSQLiteAuthorizationPredicateKeyUsesFullBinaryValues(t *testing.T) {
	ctx := context.Background()
	db := openSQLite(t, "predicate_binary_key")
	defer db.Close()
	if err := ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	pathX, pathY := strings.Repeat("a", 999)+"x", strings.Repeat("a", 999)+"y"
	for _, row := range []struct{ name, path, typeName string }{
		{"path-x", pathX, "com.example.Café"},
		{"path-y", pathY, "com.example.Café"},
		{"case-lower", pathX, "com.example.café"},
	} {
		if _, err := db.ExecContext(ctx, `INSERT INTO authorization_predicates(name,title,package_path,type_name,owner_id,status,etag,created_at,updated_at) VALUES(?,?,?,?,'test','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, row.name, row.name, row.path, row.typeName); err != nil {
			t.Fatalf("binary key rejected distinct path/type: %+v err=%v", row, err)
		}
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM authorization_predicates WHERE package_path=?`, pathX).Scan(&count); err != nil || count != 2 {
		t.Fatalf("full package path/case-distinct type rows=%d err=%v", count, err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO authorization_predicates(name,title,package_path,type_name,owner_id,status,etag,created_at,updated_at) VALUES('duplicate','Duplicate',?,'com.example.Café','test','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, pathX); err == nil {
		t.Fatal("exact duplicate binary key was accepted")
	}
}

func assertTablePresence(t *testing.T, ctx context.Context, db *sql.DB, table string, want bool) {
	t.Helper()
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if (count == 1) != want {
		t.Fatalf("table %s present=%t want=%t", table, count == 1, want)
	}
}

func openSQLite(t *testing.T, name string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open SQLite %s: %v", name, err)
	}
	db.SetMaxOpenConns(1)
	if _, err = db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		t.Fatalf("enable SQLite foreign keys: %v", err)
	}
	return db
}

func assertCount(t *testing.T, db *sql.DB, query string, want int) {
	t.Helper()
	var actual int
	if err := db.QueryRow(query).Scan(&actual); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	if actual != want {
		t.Fatalf("query %q count = %d, want %d", query, actual, want)
	}
}
