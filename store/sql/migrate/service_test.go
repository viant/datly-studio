package migrate

import (
	"context"
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
	"testing"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable foreign keys error = %v", err)
	}
	return db
}

func assertTableExists(t *testing.T, ctx context.Context, db *sql.DB, table string) {
	t.Helper()
	var name string
	err := db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
	if err != nil {
		t.Fatalf("table %s not found: %v", table, err)
	}
}

func assertTableMissing(t *testing.T, ctx context.Context, db *sql.DB, table string) {
	t.Helper()
	var name string
	err := db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
	if err == nil {
		t.Fatalf("table %s unexpectedly exists", table)
	}
	if err != sql.ErrNoRows {
		t.Fatalf("unexpected error checking table %s: %v", table, err)
	}
}

func assertComponentsConnectorFK(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	rows, err := db.QueryContext(ctx, "PRAGMA foreign_key_list(components)")
	if err != nil {
		t.Fatalf("foreign_key_list(components) error = %v", err)
	}
	defer rows.Close()

	var (
		id       int
		seq      int
		table    string
		from     string
		to       string
		onUpdate string
		onDelete string
		match    string
		found    bool
	)
	for rows.Next() {
		if err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			t.Fatalf("scan foreign key row error = %v", err)
		}
		if table == "connectors" && from == "default_connector_name" && to == "name" {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate foreign_key_list(components) error = %v", err)
	}
	if !found {
		t.Fatalf("components.default_connector_name foreign key to connectors.name not found")
	}
}

func TestReportPublicationsRejectMissingVersion(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openTestDB(t)

	service, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := service.Up(ctx, db); err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	if _, err := db.ExecContext(ctx, `
	INSERT INTO connectors(name, driver, owner_id, status, etag, created_at, updated_at)
VALUES ('analytics', 'sqlite', 'system', 'active', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("insert connector error = %v", err)
	}
	if _, err := db.ExecContext(ctx, `
	INSERT INTO namespaces(owner_id, name, title, status, etag, created_at, updated_at)
VALUES ('system', 'general', 'General', 'active', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("insert namespace error = %v", err)
	}
	if _, err := db.ExecContext(ctx, `
	INSERT INTO components(id, slug, title, owner_id, status, default_connector_name, component_scope, component_name, etag, created_at, updated_at)
VALUES ('r1', 'r1', 'Report 1', 'system', 'draft', 'analytics', 'reports', 'r1', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("insert report error = %v", err)
	}

	_, err = db.ExecContext(ctx, `
INSERT INTO component_publications(report_id, active_version_no, runtime_revision, published_by, published_at)
VALUES ('r1', 99, 'rev-1', 'system', CURRENT_TIMESTAMP)`)
	if err == nil {
		t.Fatalf("expected missing version foreign key violation")
	}
}
