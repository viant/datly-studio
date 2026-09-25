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
