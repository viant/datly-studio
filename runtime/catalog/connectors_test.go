package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
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

func TestConnectorCatalogActiveRowsOrderAndProjection(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := openCatalogTestDB(t)
	migrateCatalog(t, ctx, db)

	base := time.Date(2026, 3, 13, 12, 0, 0, 0, time.UTC)
	insert := func(name, status string, updated time.Time, deleted bool) {
		t.Helper()
		var deletedAt any
		if deleted {
			deletedAt = updated
		}
		_, err := db.ExecContext(ctx, `INSERT INTO connectors
(name, driver, owner_id, status, etag, created_at, updated_at, deleted_at)
VALUES (?, 'sqlite', 'alice', ?, 1, ?, ?, ?)`, name, status, base, updated, deletedAt)
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := range 130 {
		insert(fmt.Sprintf("connector-%03d", i), "active", base, false)
	}
	insert("first", "active", base.Add(time.Hour), false)
	insert("second", "active", base.Add(time.Hour), false)
	insert("deleted-active", "active", base.Add(2*time.Hour), true)
	insert("disabled", "disabled", base.Add(2*time.Hour), false)
	insert("draft", "draft", base.Add(2*time.Hour), false)

	rows, err := NewConnectorCatalog(db).LoadActiveConnectors(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 132 {
		t.Fatalf("active row count = %d, want 132", len(rows))
	}
	wantNames := []string{"first", "second"}
	for i := range 130 {
		wantNames = append(wantNames, fmt.Sprintf("connector-%03d", i))
	}
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row.Name)
		if row.Status != "active" {
			t.Fatalf("returned connector %q with status %q", row.Name, row.Status)
		}
	}
	if !reflect.DeepEqual(names, wantNames) {
		t.Fatalf("active connector order = %v, want %v", names, wantNames)
	}
}

func TestConnectorCatalogNullableProjectionAndCredentialSerialization(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := openCatalogTestDB(t)
	migrateCatalog(t, ctx, db)

	created := time.Date(2026, 3, 13, 12, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)
	tested := created.Add(30 * time.Minute)
	_, err := db.ExecContext(ctx, `INSERT INTO connectors
(name, driver, dsn_template, secret_ref, description, owner_id, status, options_json,
 last_test_status, last_test_error_code, last_tested_at, etag, created_at, updated_at)
VALUES ('complete', 'sqlite', 'file:private?mode=memory', 'secret://private', 'A description',
 'alice', 'active', '{"pool_size":"small"}', 'failed', 'TEST_FAILED', ?, 7, ?, ?)`, tested, created, updated)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO connectors
(name, driver, dsn_template, owner_id, status, etag, created_at, updated_at)
VALUES ('nullable', 'sqlite', '  ', 'bob', 'active', 2, ?, ?)`, created, created)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO connectors
(name, driver, owner_id, status, etag, created_at, updated_at)
VALUES ('null-dsn', 'sqlite', 'carol', 'active', 3, ?, ?)`, created, created)
	if err != nil {
		t.Fatal(err)
	}

	rows, err := NewConnectorCatalog(db).LoadActiveConnectors(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].Name != "complete" || rows[1].Name != "null-dsn" || rows[2].Name != "nullable" {
		t.Fatalf("unexpected connector count or order: got %d rows", len(rows))
	}
	complete := rows[0]
	if complete.Driver != "sqlite" || complete.DSNTemplate != "file:private?mode=memory" || !complete.DSNConfigured ||
		complete.SecretRef != "secret://private" || complete.Description != "A description" || complete.OwnerID != "alice" ||
		complete.Status != "active" || complete.LastTestStatus != "failed" || complete.LastTestErrorCode != "TEST_FAILED" ||
		complete.ETag != 7 || !complete.CreatedAt.Equal(created) || !complete.UpdatedAt.Equal(updated) ||
		complete.LastTestedAt == nil || !complete.LastTestedAt.Equal(tested) || string(complete.Options) != `{"pool_size":"small"}` {
		t.Fatal("full connector projection changed")
	}
	redacted, err := json.Marshal(complete)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(redacted), "file:private") || strings.Contains(string(redacted), "secret://private") {
		t.Fatalf("serialized connector leaked connection material: %s", redacted)
	}
	if rows[1].DSNTemplate != "" || rows[1].DSNConfigured || rows[1].Options != nil {
		t.Fatal("NULL DSN or options projection changed")
	}
	nullable := rows[2]
	if nullable.DSNTemplate != "  " || nullable.DSNConfigured || nullable.SecretRef != "" || nullable.Description != "" ||
		nullable.Options != nil || nullable.LastTestStatus != "" || nullable.LastTestErrorCode != "" || nullable.LastTestedAt != nil {
		t.Fatal("nullable projection changed")
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
