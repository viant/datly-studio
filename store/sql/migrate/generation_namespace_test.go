package migrate

import (
	"context"

	"testing"
)

func TestLegacyGenerationOwnershipRequiresCompleteSingleNamespaceReferences(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	for _, statement := range []string{
		`CREATE TABLE schema_version(version INTEGER); INSERT INTO schema_version VALUES(17)`,
		`CREATE TABLE components(id TEXT PRIMARY KEY,namespace_id TEXT NOT NULL); INSERT INTO components VALUES('a','alpha'),('a2','alpha'),('b','beta')`,
		`CREATE TABLE runtime_generations(generation_no INTEGER PRIMARY KEY,report_count INTEGER,namespace_id TEXT NOT NULL DEFAULT ''); INSERT INTO runtime_generations VALUES(1,2,''),(2,2,''),(3,3,''),(4,1,''),(5,1,'existing'),(6,1,'')`,
		`CREATE TABLE report_publications(report_id TEXT,active_generation INTEGER,desired_generation INTEGER); INSERT INTO report_publications VALUES('a',1,2),('a2',1,3),('b',2,2),('orphan',6,6)`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := migrateLegacyGenerationOwnership(ctx, db); err != nil {
		t.Fatal(err)
	}
	for id, expected := range map[int]string{1: "alpha", 2: "", 3: "", 4: "", 5: "existing", 6: ""} {
		var actual string
		if err := db.QueryRowContext(ctx, `SELECT namespace_id FROM runtime_generations WHERE generation_no=?`, id).Scan(&actual); err != nil || actual != expected {
			t.Fatalf("generation %d ownership=%q want=%q err=%v", id, actual, expected, err)
		}
	}
	var version int
	if err := db.QueryRowContext(ctx, `SELECT version FROM schema_version`).Scan(&version); err != nil || version != 19 {
		t.Fatalf("schema version=%d err=%v", version, err)
	}
	if err := migrateLegacyGenerationOwnership(ctx, db); err != nil {
		t.Fatal(err)
	}
}
