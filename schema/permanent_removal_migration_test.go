package schema

import (
	"context"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"os"
	"strings"
	"testing"
)

func TestPermanentRemovalMigration28(t *testing.T) {
	ctx := context.Background()
	db := openSQLite(t, "removal_migration")
	defer db.Close()
	ddl, err := canonicalDDL("sqlite")
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := tableStatement(ddl, "component_deletions")
	if err != nil {
		t.Fatal(err)
	}
	ddl = strings.Replace(ddl, receipt, "", 1)
	ddl = strings.Replace(ddl, "    last_version_no         INT NOT NULL DEFAULT 0,\n", "", 1)
	oldFK := `    CONSTRAINT fk_component_publication_events_report
        FOREIGN KEY (report_id) REFERENCES components(id) ON DELETE CASCADE,
    CONSTRAINT fk_component_publication_events_version
        FOREIGN KEY (report_id, version_no) REFERENCES component_versions(report_id, version_no),
`
	ddl = strings.Replace(ddl, "    CONSTRAINT fk_component_publication_events_generation", oldFK+"    CONSTRAINT fk_component_publication_events_generation", 1)
	if _, err = db.ExecContext(ctx, ddl); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`PRAGMA foreign_keys=ON`,
		`INSERT INTO connectors(name,driver,owner_id,status,created_at,updated_at) VALUES('test','sqlite','owner','active',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO namespaces(owner_id,name,title,status,created_at,updated_at) VALUES('owner','general','General','active',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO components(id,slug,title,owner_id,status,default_connector_name,component_scope,component_name,created_at,updated_at) VALUES('c1','one','One','owner','active','test','example/one','one',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),('c2','two','Two','owner','draft','test','example/two','two',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO component_versions(report_id,version_no,state,authoring_mode,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,created_by,created_at) VALUES('c1',7,'draft','sql','{}','1','hash','{}','pending','test','test','owner',CURRENT_TIMESTAMP)`,
		`INSERT INTO component_publication_events(event_id,report_id,owner_id,operation,version_no,status,requested_by,occurred_at) VALUES('e1','c1','owner','publish',7,'failed','owner',CURRENT_TIMESTAMP)`,
	} {
		if _, err = db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if err = Initialize(ctx, db, "sqlite"); err == nil {
		t.Fatal("old storage accepted")
	}
	script, err := os.ReadFile("migrations/028-permanent-removal.sqlite.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, string(script)); err != nil {
		t.Fatal(err)
	}
	if err = Verify(ctx, db, "sqlite"); err != nil {
		t.Fatal(err)
	}
	assertCount(t, db, "SELECT last_version_no FROM components WHERE id='c1'", 7)
	assertCount(t, db, "SELECT last_version_no FROM components WHERE id='c2'", 0)
	if _, err = db.ExecContext(ctx, `INSERT INTO component_deletions VALUES('c1','namespace','owner',CURRENT_TIMESTAMP,'owner'); DELETE FROM components WHERE id='c1'`); err != nil {
		t.Fatal(err)
	}
	assertCount(t, db, "SELECT COUNT(*) FROM component_publication_events WHERE event_id='e1' AND version_no=7", 1)
	assertCount(t, db, "SELECT COUNT(*) FROM component_deletions WHERE component_id='c1'", 1)
	if _, err = db.ExecContext(ctx, `UPDATE component_publication_events SET generation_no=999 WHERE event_id='e1'`); err == nil {
		t.Fatal("generation FK removed")
	}
	if _, err = db.ExecContext(ctx, `INSERT INTO component_deletions VALUES('c1','other','other',CURRENT_TIMESTAMP,'other')`); err == nil {
		t.Fatal("receipt identity reused")
	}
}

func TestInitializeRejectsRemovalConstraintDrift(t *testing.T) {
	for _, test := range []struct{ name, old, new string }{
		{"counter_nullable", "last_version_no         INT NOT NULL DEFAULT 0", "last_version_no         INT DEFAULT 0"},
		{"counter_default", "last_version_no         INT NOT NULL DEFAULT 0", "last_version_no         INT NOT NULL DEFAULT 1"},
		{"generation_removed", "    CONSTRAINT fk_component_publication_events_generation\n        FOREIGN KEY (generation_no) REFERENCES runtime_generations(generation_no),\n", ""},
		{"component_fk", "    CONSTRAINT fk_component_publication_events_generation", "    FOREIGN KEY (report_id) REFERENCES components(id) ON DELETE CASCADE,\n    CONSTRAINT fk_component_publication_events_generation"},
		{"receipt_fk", "    PRIMARY KEY (component_id)", "    PRIMARY KEY (component_id),\n    FOREIGN KEY (component_id) REFERENCES components(id)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := openSQLite(t, "drift_"+test.name)
			defer db.Close()
			ddl, err := canonicalDDL("sqlite")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(ddl, test.old) {
				t.Fatal("missing test target")
			}
			if _, err = db.ExecContext(context.Background(), strings.Replace(ddl, test.old, test.new, 1)); err != nil {
				t.Fatal(err)
			}
			if err = Initialize(context.Background(), db, "sqlite"); err == nil {
				t.Fatal("constraint drift accepted")
			}
		})
	}
}

// This test requires an isolated, empty database owned by the caller.
func TestPermanentRemovalMigration28MySQL(t *testing.T) {
	dsn := os.Getenv("STUDIO_REMOVAL_MYSQL_DSN")
	if dsn == "" {
		t.Skip("isolated MySQL fixture is required")
	}
	ctx := context.Background()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ddl, err := MySQLDDL()
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := tableStatement(ddl, "component_deletions")
	if err != nil {
		t.Fatal(err)
	}
	ddl = strings.Replace(ddl, receipt, "", 1)
	ddl = strings.Replace(ddl, "    last_version_no         INT NOT NULL DEFAULT 0,\n", "", 1)
	ddl = strings.Replace(ddl, "    CONSTRAINT fk_component_publication_events_generation", `    CONSTRAINT fk_component_publication_events_report
        FOREIGN KEY (report_id) REFERENCES components(id) ON DELETE CASCADE,
    CONSTRAINT fk_component_publication_events_version
        FOREIGN KEY (report_id, version_no) REFERENCES component_versions(report_id, version_no),
    CONSTRAINT fk_component_publication_events_generation`, 1)
	for _, statement := range mysqlStatements(ddl) {
		if _, err = db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	for _, statement := range []string{
		`INSERT INTO connectors(name,driver,owner_id,status,created_at,updated_at) VALUES('test','mysql','owner','active',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO namespaces(owner_id,name,title,status,created_at,updated_at) VALUES('owner','general','General','active',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO components(id,slug,title,owner_id,status,default_connector_name,component_scope,component_name,created_at,updated_at) VALUES('c1','one','One','owner','active','test','example/one','one',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),('c2','two','Two','owner','draft','test','example/two','two',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO component_versions(report_id,version_no,state,authoring_mode,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,created_by,created_at) VALUES('c1',7,'draft','sql','{}','1','hash','{}','pending','test','test','owner',CURRENT_TIMESTAMP)`,
		`INSERT INTO component_publication_events(event_id,report_id,owner_id,operation,version_no,status,requested_by,occurred_at) VALUES('e1','c1','owner','publish',7,'failed','owner',CURRENT_TIMESTAMP)`,
	} {
		if _, err = db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if err = Initialize(ctx, db, "mysql"); err == nil {
		t.Fatal("old layout accepted")
	}
	script, err := os.ReadFile("migrations/028-permanent-removal.mysql.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range mysqlStatements(string(script)) {
		if _, err = db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if err = Verify(ctx, db, "mysql"); err != nil {
		t.Fatal(err)
	}
	assertCount(t, db, "SELECT last_version_no FROM components WHERE id='c1'", 7)
	assertCount(t, db, "SELECT last_version_no FROM components WHERE id='c2'", 0)
	for _, statement := range []string{`INSERT INTO component_deletions VALUES('c1','namespace','owner',CURRENT_TIMESTAMP,'owner')`, `DELETE FROM components WHERE id='c1'`} {
		if _, err = db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	assertCount(t, db, "SELECT COUNT(*) FROM component_publication_events WHERE event_id='e1' AND version_no=7", 1)
	if _, err = db.ExecContext(ctx, `UPDATE component_publication_events SET generation_no=999 WHERE event_id='e1'`); err == nil {
		t.Fatal("generation FK removed")
	}
	// Existing stores are checked read-only, including exact removed FK ownership.
	if _, err = db.ExecContext(ctx, `ALTER TABLE component_publication_events ADD CONSTRAINT unexpected_component_fk FOREIGN KEY(report_id) REFERENCES components(id)`); err == nil {
		t.Fatal("fixture should contain retained history with no live component")
	}
	if _, err = db.ExecContext(ctx, `ALTER TABLE components MODIFY last_version_no INT NULL DEFAULT 0`); err != nil {
		t.Fatal(err)
	}
	if err = Verify(ctx, db, "mysql"); err == nil {
		t.Fatal("nullable allocation counter accepted")
	}
}
