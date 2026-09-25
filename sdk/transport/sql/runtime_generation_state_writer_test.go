package sqltransport

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	stored "github.com/viant/datly-studio/studio/runtime_generations/store_state"
	xhandler "github.com/viant/xdatly/handler"
	_ "modernc.org/sqlite"
)

func TestGenerationStateWriterActivatesBuildingRow(t *testing.T) {
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
	now := time.Now().UTC()
	if err := transport.insertBuildingGeneration(ctx, nil, 1, "rev1", "owner", now); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	count := 2
	row := &stored.StoredGeneration{GenerationNo: 1, Status: "building", ReportCount: &count, ActivatedAt: &now,
		Has: &stored.StoredGenerationHas{GenerationNo: true, Status: true, ReportCount: true, ActivatedAt: true}}
	if err := transport.writeGenerationState(ctx, tx, "activate", 1, []*stored.StoredGeneration{row}); err != nil {
		t.Fatalf("generated activation error (%T): %+v", err, err)
	}
	stale := &stored.StoredGeneration{GenerationNo: 1, Status: "building", ReportCount: &count, ActivatedAt: &now,
		Has: &stored.StoredGenerationHas{GenerationNo: true, Status: true, ReportCount: true, ActivatedAt: true}}
	var conflict *xhandler.Conflict
	if err := transport.writeGenerationState(ctx, tx, "activate", 1, []*stored.StoredGeneration{stale}); !errors.As(err, &conflict) {
		t.Fatalf("stale building token error=%v", err)
	}
}

func TestGenerationStateWriterFailsBuildingRow(t *testing.T) {
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
	now := time.Now().UTC()
	if err := transport.insertBuildingGeneration(ctx, nil, 1, "rev1", "owner", now); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	diagnostics := `[{"code":"expired"}]`
	row := &stored.StoredGeneration{GenerationNo: 1, Status: "building", DiagnosticsJson: &diagnostics, RetiredAt: &now,
		Has: &stored.StoredGenerationHas{GenerationNo: true, Status: true, DiagnosticsJson: true, RetiredAt: true}}
	if err := transport.writeGenerationState(ctx, tx, "fail", 1, []*stored.StoredGeneration{row}); err != nil {
		t.Fatal(err)
	}
	var status, storedDiagnostics string
	var retiredAt sql.NullTime
	if err := tx.QueryRowContext(ctx, `SELECT status,diagnostics_json,retired_at FROM runtime_generations WHERE generation_no=1`).
		Scan(&status, &storedDiagnostics, &retiredAt); err != nil || status != "failed" || storedDiagnostics != diagnostics || !retiredAt.Valid {
		t.Fatalf("failed generation status=%q diagnostics=%q retired=%v err=%v", status, storedDiagnostics, retiredAt, err)
	}
	stale := &stored.StoredGeneration{GenerationNo: 1, Status: "building", DiagnosticsJson: &diagnostics,
		Has: &stored.StoredGenerationHas{GenerationNo: true, Status: true, DiagnosticsJson: true}}
	var conflict *xhandler.Conflict
	if err := transport.writeGenerationState(ctx, tx, "fail", 1, []*stored.StoredGeneration{stale}); !errors.As(err, &conflict) {
		t.Fatalf("stale failure token error=%v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	var restored string
	if err := db.QueryRowContext(ctx, `SELECT status FROM runtime_generations WHERE generation_no=1`).Scan(&restored); err != nil || restored != "building" {
		t.Fatalf("rolled-back generation status=%q err=%v", restored, err)
	}
}
