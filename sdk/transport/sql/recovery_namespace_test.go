package sqltransport

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	_ "modernc.org/sqlite"
)

func TestExpiredGenerationRecoveryDoesNotBlockOrFailAnotherNamespace(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db}
	a := sdk.WithNamespaceSelection(ctx, namespaceaccess.ID("owner", "alpha"))
	b := sdk.WithNamespaceSelection(ctx, namespaceaccess.ID("owner", "beta"))
	now := time.Now().UTC()
	if err := transport.insertBuildingGeneration(a, nil, 1, "stale-a", "owner", now.Add(-stagedGenerationLease-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := transport.insertBuildingGeneration(b, nil, 2, "fresh-b", "owner", now); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err := transport.ensureNoStagedGeneration(a, tx, now); err != nil {
		t.Fatalf("namespace B blocked A recovery: %v", err)
	}
	for number, want := range map[int64]string{1: "failed", 2: "building"} {
		var status string
		if err := tx.QueryRowContext(ctx, `SELECT status FROM runtime_generations WHERE generation_no=?`, number).Scan(&status); err != nil || status != want {
			t.Fatalf("generation %d status=%s want=%s err=%v", number, status, want, err)
		}
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	for number := range map[int64]bool{1: true, 2: true} {
		var status string
		if err := db.QueryRowContext(ctx, `SELECT status FROM runtime_generations WHERE generation_no=?`, number).Scan(&status); err != nil || status != "building" {
			t.Fatalf("recovery rollback generation %d status=%s err=%v", number, status, err)
		}
	}
}
