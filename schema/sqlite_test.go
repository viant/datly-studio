package schema

import (
	"context"
	"database/sql"
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
	assertCount(t, studio, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'", 23)
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
