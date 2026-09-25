package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	_ "modernc.org/sqlite"
)

func TestConnectorCatalogReaderPaginatesBeyondDefaultViewLimit(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	const count = 137
	for i := count - 1; i >= 0; i-- {
		name := fmt.Sprintf("connector-%03d", i)
		var dsn, secret any
		if i == 0 {
			dsn, secret = "file:primary.db", "secret://primary"
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO connectors
			(name,driver,dsn_template,secret_ref,description,owner_id,status,options_json,etag,created_at,updated_at)
			VALUES(?,?,?,?,?,?,?,?,?,?,?)`, name, "sqlite", dsn, secret, nil, "owner", "draft", `{}`, 1, now, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db}
	owner := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner"})
	items, err := transport.readConnectorCatalog(owner, connectorCatalogRequest{Query: "SQLITE", Limit: 150})
	if err != nil || len(items) != count {
		t.Fatalf("catalog count=%d err=%v", len(items), err)
	}
	if items[0].Name != "connector-000" || items[count-1].Name != "connector-136" ||
		!items[0].DSNConfigured || !items[0].SecretConfigured || string(items[0].Options) != `{}` ||
		items[1].DSNConfigured || items[1].SecretConfigured {
		t.Fatalf("catalog ordering/nullable fields: first=%+v second=%+v last=%+v", items[0], items[1], items[count-1])
	}
	page, err := transport.readConnectorCatalog(owner, connectorCatalogRequest{Limit: 1, Offset: 136})
	if err != nil || len(page) != 1 || page[0].Name != "connector-136" {
		t.Fatalf("tail page=%+v err=%v", page, err)
	}
	other := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "other"})
	hidden, err := transport.readConnectorCatalog(other, connectorCatalogRequest{Limit: 150})
	if err != nil || len(hidden) != 0 {
		t.Fatalf("other principal catalog=%d err=%v", len(hidden), err)
	}
}

func TestAvailableConnectorNamesPaginatesGeneratedCatalog(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	const count = 503
	for i := count - 1; i >= 0; i-- {
		name := fmt.Sprintf("connector-%03d", i)
		if _, err := tx.ExecContext(ctx, `INSERT INTO connectors
			(name,driver,owner_id,status,options_json,etag,created_at,updated_at)
			VALUES(?,?,?,?,?,?,?,?)`, name, "sqlite", "owner", "active", `{}`, 1, now, now); err != nil {
			t.Fatal(err)
		}
	}
	for _, extra := range []struct{ name, owner, status string }{
		{"outside", "other", "active"}, {"inactive", "owner", "draft"},
	} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO connectors
			(name,driver,owner_id,status,options_json,etag,created_at,updated_at)
			VALUES(?,?,?,?,?,?,?,?)`, extra.name, "sqlite", extra.owner, extra.status, `{}`, 1, now, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db}
	owner := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner"})
	names, err := transport.availableConnectorNames(owner)
	if err != nil || len(names) != count {
		t.Fatalf("owner active connector names: count=%d err=%v", len(names), err)
	}
	if names[0] != "connector-000" || names[count-1] != "connector-502" {
		t.Fatalf("owner active connector ordering: first=%s last=%s", names[0], names[count-1])
	}
	other := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "other"})
	names, err = transport.availableConnectorNames(other)
	if err != nil || len(names) != 1 || names[0] != "outside" {
		t.Fatalf("other active connector names=%v err=%v", names, err)
	}
}
