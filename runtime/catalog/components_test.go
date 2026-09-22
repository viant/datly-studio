package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/store/sql/fixture"
	_ "modernc.org/sqlite"
)

func TestComponentCatalogLoadsCanonicalPublishedReport(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatal(err)
	}
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	seed, err := fixture.SeedSampleCatalog(ctx, db, time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	components, err := NewComponentCatalog(db).LoadPublishedComponents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(components) != 1 {
		t.Fatalf("published reports = %d, want 1", len(components))
	}
	got := components[0]
	if got.Report.ID != seed.PublishedComponentID || got.Version.VersionNo != 1 || got.Version.AuthoringMode != "dql" {
		t.Fatalf("unexpected canonical published report: %+v", got)
	}
	if got.Publication.RuntimeRevision != "rev-0001" {
		t.Fatalf("runtime revision = %q, want rev-0001", got.Publication.RuntimeRevision)
	}
}
