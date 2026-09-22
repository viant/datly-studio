package fixture

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	_ "modernc.org/sqlite"
)

func TestSeedSampleCatalog(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable foreign keys error = %v", err)
	}

	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatalf("schema.ApplySQLite() error = %v", err)
	}

	now := time.Date(2026, 3, 13, 12, 0, 0, 0, time.UTC)
	seed, err := SeedSampleCatalog(ctx, db, now)
	if err != nil {
		t.Fatalf("SeedSampleCatalog() error = %v", err)
	}

	assertCount(t, ctx, db, "SELECT COUNT(1) FROM connectors", 2)
	assertCount(t, ctx, db, "SELECT COUNT(1) FROM reports", 2)
	assertCount(t, ctx, db, "SELECT COUNT(1) FROM report_versions", 2)
	assertCount(t, ctx, db, "SELECT COUNT(1) FROM report_fields", 4)
	assertCount(t, ctx, db, "SELECT COUNT(1) FROM report_parameters", 2)
	assertCount(t, ctx, db, "SELECT COUNT(1) FROM report_predicates", 2)
	assertCount(t, ctx, db, "SELECT COUNT(1) FROM report_acl", 4)
	assertCount(t, ctx, db, "SELECT COUNT(1) FROM report_publications", 1)
	assertCount(t, ctx, db, "SELECT COUNT(1) FROM report_views", 2)
	assertCount(t, ctx, db, "SELECT COUNT(1) FROM report_cube_configs", 1)
	assertCount(t, ctx, db, "SELECT COUNT(1) FROM report_mcp_exposures", 1)
	assertCount(t, ctx, db, "SELECT COUNT(1) FROM runtime_generations", 1)

	var primaryConnectorCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(1) FROM connectors WHERE name=?", seed.PrimaryConnectorName).Scan(&primaryConnectorCount); err != nil {
		t.Fatalf("count primary connectors error = %v", err)
	}
	if primaryConnectorCount != 1 {
		t.Fatalf("primary connector count = %d, want 1", primaryConnectorCount)
	}

	var sqlText, dqlText string
	if err := db.QueryRowContext(ctx, "SELECT authored_sql, authored_dql FROM report_versions WHERE report_id=? AND version_no=1", seed.DraftComponentID).Scan(&sqlText, &dqlText); err != nil {
		t.Fatalf("load draft component version error = %v", err)
	}
	if sqlText == "" || dqlText == "" {
		t.Fatalf("expected both SQL and DQL to be seeded, got sql=%q dql=%q", sqlText, dqlText)
	}

	var activeVersion int
	if err := db.QueryRowContext(ctx, "SELECT active_version_no FROM report_publications WHERE report_id=?", seed.PublishedComponentID).Scan(&activeVersion); err != nil {
		t.Fatalf("load publication error = %v", err)
	}
	if activeVersion != 1 {
		t.Fatalf("active publication version = %d, want 1", activeVersion)
	}

	var publishedStatus string
	if err := db.QueryRowContext(ctx, "SELECT status FROM reports WHERE id=?", seed.PublishedComponentID).Scan(&publishedStatus); err != nil {
		t.Fatalf("load published report status error = %v", err)
	}
	if publishedStatus != "active" {
		t.Fatalf("published report status = %q, want %q", publishedStatus, "active")
	}

	var publicationStatus string
	if err := db.QueryRowContext(ctx, "SELECT publication_status FROM report_publications WHERE report_id=?", seed.PublishedComponentID).Scan(&publicationStatus); err != nil {
		t.Fatalf("load publication status error = %v", err)
	}
	if publicationStatus != "active" {
		t.Fatalf("publication status = %q, want %q", publicationStatus, "active")
	}

	var advancedAllowed int
	if err := db.QueryRowContext(ctx, `
SELECT can_use_dql
FROM report_acl
WHERE report_id=? AND subject_type='role' AND subject_id='report_admin'`, seed.PublishedComponentID).Scan(&advancedAllowed); err != nil {
		t.Fatalf("load report admin ACL error = %v", err)
	}
	if advancedAllowed != 1 {
		t.Fatalf("report admin DQL permission = %d, want 1", advancedAllowed)
	}
}

func assertCount(t *testing.T, ctx context.Context, db *sql.DB, query string, expected int) {
	t.Helper()
	var actual int
	if err := db.QueryRowContext(ctx, query).Scan(&actual); err != nil {
		t.Fatalf("count query failed for %q: %v", query, err)
	}
	if actual != expected {
		t.Fatalf("count for %q = %d, want %d", query, actual, expected)
	}
}
