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

func TestGenerationHeadSeesCallerTransactionAndRollback(t *testing.T) {
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
	next, err := transport.nextGenerationNo(ctx, nil)
	if err != nil || next != 1 {
		t.Fatalf("initial generation=%d err=%v", next, err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `INSERT INTO runtime_generations(generation_no,source_revision,status,report_count,build_manifest_json,requested_by,requested_at)
		VALUES(1,'rev-1','building',0,'{}','owner',?)`, now); err != nil {
		t.Fatal(err)
	}
	next, err = transport.nextGenerationNo(ctx, tx)
	if err != nil || next != 2 {
		t.Fatalf("uncommitted generation head=%d err=%v", next, err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	next, err = transport.nextGenerationNo(ctx, nil)
	if err != nil || next != 1 {
		t.Fatalf("rolled-back generation head=%d err=%v", next, err)
	}
}
