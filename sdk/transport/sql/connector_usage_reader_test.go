package sqltransport

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	_ "modernc.org/sqlite"
)

func TestConnectorUsageReaderCountsOnlyLiveReports(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db}
	used, err := transport.connectorUsage(ctx, "main")
	if err != nil || used != 0 {
		t.Fatalf("empty usage=%d err=%v", used, err)
	}
	now := time.Now().UTC()
	if _, err := db.ExecContext(ctx, `INSERT INTO connectors(name,driver,owner_id,status,etag,created_at,updated_at)
		VALUES('main','sqlite','owner','active',1,?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at)
		VALUES('owner','general','General','active',1,?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO reports(id,namespace,slug,title,owner_id,status,
		default_connector_name,component_scope,component_name,etag,created_at,updated_at)
		VALUES('report','general','report','Report','owner','draft','main','reports','reader',1,?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	used, err = transport.connectorUsage(ctx, "main")
	if err != nil || used != 1 {
		t.Fatalf("live usage=%d err=%v", used, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE reports SET deleted_at=? WHERE id='report'`, now); err != nil {
		t.Fatal(err)
	}
	used, err = transport.connectorUsage(ctx, "main")
	if err != nil || used != 0 {
		t.Fatalf("soft-deleted usage=%d err=%v", used, err)
	}
}
