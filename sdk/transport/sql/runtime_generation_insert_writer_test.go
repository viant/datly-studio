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

func TestGenerationInsertWriterJoinsCallerTransaction(t *testing.T) {
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
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	if err := transport.insertBuildingGeneration(ctx, tx, 1, "report:1:1", "owner", now); err != nil {
		t.Fatal(err)
	}
	next, err := transport.nextGenerationNo(ctx, tx)
	if err != nil || next != 2 {
		t.Fatalf("transaction generation=%d err=%v", next, err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	next, err = transport.nextGenerationNo(ctx, nil)
	if err != nil || next != 1 {
		t.Fatalf("rolled-back generation=%d err=%v", next, err)
	}
	if err := transport.insertBuildingGeneration(ctx, nil, 1, "report:1:1", "owner", now); err != nil {
		t.Fatal(err)
	}
	if err := transport.insertBuildingGeneration(ctx, nil, 1, "report:1:1", "owner", now); err == nil {
		t.Fatal("duplicate generation identity was accepted")
	}
}
