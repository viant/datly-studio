package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/viant/datly-studio/store/sql/migrate"
	_ "modernc.org/sqlite"
)

func TestConnectorCatalogLoadActiveConnectors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openCatalogTestDB(t)
	migrateCatalog(t, ctx, db)

	now := time.Date(2026, 3, 13, 12, 0, 0, 0, time.UTC)
	if _, err := db.ExecContext(ctx, `INSERT INTO connectors(name, driver, dsn_template, secret_ref, owner_id, status, options_json, etag, created_at, updated_at) VALUES ('analytics', 'sqlite', 'file:analytics?mode=memory&cache=shared', 'secret://analytics', 'alice', 'active', '{"pool_size":"small"}', 1, ?, ?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO connectors(name, driver, dsn_template, secret_ref, owner_id, status, etag, created_at, updated_at) VALUES ('warehouse', 'sqlite', 'file:warehouse?mode=memory&cache=shared', 'secret://warehouse', 'bob', 'disabled', 1, ?, ?)`, now, now); err != nil {
		t.Fatal(err)
	}

	catalog := NewConnectorCatalog(db)
	connectors, err := catalog.LoadActiveConnectors(ctx)
	if err != nil {
		t.Fatalf("LoadActiveConnectors() error = %v", err)
	}
	if len(connectors) != 1 {
		t.Fatalf("LoadActiveConnectors() len = %d, want 1", len(connectors))
	}
	if connectors[0].Name != "analytics" {
		t.Fatalf("LoadActiveConnectors() returned %q, want analytics", connectors[0].Name)
	}
}

func openCatalogTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable foreign keys error = %v", err)
	}
	return db
}

func migrateCatalog(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	service, err := migrate.New()
	if err != nil {
		t.Fatalf("migrate.New() error = %v", err)
	}
	if err := service.Up(ctx, db); err != nil {
		t.Fatalf("migrate.Up() error = %v", err)
	}
}
