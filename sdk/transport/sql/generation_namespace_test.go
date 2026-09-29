package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	generationstate "github.com/viant/datly-studio/studio/runtime_generations/store_state"
	_ "modernc.org/sqlite"
)

func TestGenerationActivationRetiresOnlySelectedNamespace(t *testing.T) {
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
	aID, bID := namespaceaccess.ID("owner", "alpha"), namespaceaccess.ID("owner", "beta")
	a, b := sdk.WithNamespaceSelection(ctx, aID), sdk.WithNamespaceSelection(ctx, bID)
	now := time.Now().UTC()
	for _, item := range []struct {
		ctx    context.Context
		number int64
	}{{a, 1}, {b, 2}, {a, 3}} {
		if err := transport.insertBuildingGeneration(item.ctx, nil, item.number, fmt.Sprintf("generation-%d", item.number), "owner", now); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`UPDATE runtime_generations SET status='active' WHERE generation_no IN (1,2)`); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err := transport.activateGenerationState(a, tx, 3, 1, now); err != nil {
		t.Fatal(err)
	}
	for number, want := range map[int64]string{1: "retired", 2: "active", 3: "active"} {
		var status, namespaceID string
		if err := tx.QueryRowContext(ctx, `SELECT status,namespace_id FROM runtime_generations WHERE generation_no=?`, number).Scan(&status, &namespaceID); err != nil || status != want {
			t.Fatalf("generation %d status=%s want=%s err=%v", number, status, want, err)
		}
		expectedNamespace := aID
		if number == 2 {
			expectedNamespace = bID
		}
		if namespaceID != expectedNamespace {
			t.Fatalf("generation %d ownership=%q", number, namespaceID)
		}
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := db.QueryRowContext(ctx, `SELECT status FROM runtime_generations WHERE generation_no=1`).Scan(&status); err != nil || status != "active" {
		t.Fatalf("activation rollback status=%s err=%v", status, err)
	}
}

func TestGenerationMutationRejectsForeignNamespaceAndOwnershipChanges(t *testing.T) {
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
	aID, bID := namespaceaccess.ID("owner", "alpha"), namespaceaccess.ID("owner", "beta")
	a, b := sdk.WithNamespaceSelection(ctx, aID), sdk.WithNamespaceSelection(ctx, bID)
	now := time.Now().UTC()
	if err := transport.insertBuildingGeneration(a, nil, 1, "a", "owner", now); err != nil {
		t.Fatal(err)
	}
	if err := transport.insertBuildingGeneration(b, nil, 2, "b", "owner", now); err != nil {
		t.Fatal(err)
	}
	diagnostics := `[{"severity":"error","message":"fixture"}]`
	for _, attempt := range []struct {
		number       int64
		ownership    string
		setOwnership bool
	}{{2, "", false}, {1, bID, true}} {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		row := &generationstate.StoredGeneration{NamespaceId: attempt.ownership, GenerationNo: attempt.number, Status: "building", DiagnosticsJson: &diagnostics, Has: &generationstate.StoredGenerationHas{NamespaceId: attempt.setOwnership, GenerationNo: true, Status: true, DiagnosticsJson: true}}
		if err := transport.writeGenerationState(a, tx, "fail", attempt.number, []*generationstate.StoredGeneration{row}); err == nil {
			tx.Rollback()
			t.Fatal("foreign generation or namespace ownership mutation accepted")
		}
		tx.Rollback()
	}
	for number, id := range map[int64]string{1: aID, 2: bID} {
		var status, namespaceID string
		if err := db.QueryRowContext(ctx, `SELECT status,namespace_id FROM runtime_generations WHERE generation_no=?`, number).Scan(&status, &namespaceID); err != nil || status != "building" || namespaceID != id {
			t.Fatalf("denied mutation changed generation %d status=%s owner=%s err=%v", number, status, namespaceID, err)
		}
	}
}
