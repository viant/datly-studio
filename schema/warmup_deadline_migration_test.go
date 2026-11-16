package schema

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"
)

func TestWarmupDeadlineMigration29(t *testing.T) {
	ctx := t.Context()
	db := openSQLite(t, "warmup_deadline_migration")
	defer db.Close()
	ddl, err := canonicalDDL("sqlite")
	if err != nil {
		t.Fatal(err)
	}
	old := strings.Replace(ddl, "    deadline_at             DATETIME NULL,\n", "", 1)
	if old == ddl {
		t.Fatal("fixture did not remove the new column")
	}
	if _, err = db.ExecContext(ctx, old); err != nil {
		t.Fatal(err)
	}
	if err = SetSQLiteVersion(ctx, db, 28); err != nil {
		t.Fatal(err)
	}
	requested := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
	seedLegacyDeadlineRun(ctx, t, db, requested)
	if err = Initialize(ctx, db, "sqlite"); err == nil {
		t.Fatal("startup accepted layout 28")
	}
	var columns int
	if err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pragma_table_info('component_warmup_runs') WHERE name='deadline_at'").Scan(&columns); err != nil || columns != 0 {
		t.Fatalf("startup altered storage: %v", err)
	}
	script, err := os.ReadFile("migrations/029-warmup-deadline.sqlite.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(ctx, string(script)); err != nil {
		t.Fatal(err)
	}
	if err = Verify(ctx, db, "sqlite"); err != nil {
		t.Fatal(err)
	}
	assertLegacyDeadlineRun(ctx, t, db, requested)
	version, err := SQLiteVersion(ctx, db)
	if err != nil || version != 29 {
		t.Fatalf("version=%d err=%v", version, err)
	}
	var required int
	if err = db.QueryRowContext(ctx, "SELECT \"notnull\" FROM pragma_table_info('component_warmup_runs') WHERE name='deadline_at'").Scan(&required); err != nil || required != 0 {
		t.Fatalf("deadline must be nullable: %v", err)
	}
	mysql, err := os.ReadFile("migrations/029-warmup-deadline.mysql.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mysql), "ADD COLUMN deadline_at DATETIME(6) NULL") || strings.Contains(strings.ToUpper(string(mysql)), "UPDATE COMPONENT_WARMUP_RUNS") {
		t.Fatal("MySQL upgrade must add a nullable deadline without rewriting history")
	}
}

func TestWarmupDeadlineMigration29MySQL(t *testing.T) {
	dsn := os.Getenv("STUDIO_WARMUP_MYSQL_DSN")
	if dsn == "" {
		t.Skip("owned MySQL fixture required")
	}
	ctx := t.Context()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ddl, err := MySQLDDL()
	if err != nil {
		t.Fatal(err)
	}
	old := strings.Replace(ddl, "    deadline_at             DATETIME(6) NULL,\n", "", 1)
	if old == ddl {
		t.Fatal("fixture did not remove deadline")
	}
	for _, statement := range mysqlStatements(old) {
		if _, err = db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = db.ExecContext(ctx, "UPDATE schema_version SET version=28"); err != nil {
		t.Fatal(err)
	}
	requested := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
	seedLegacyDeadlineRun(ctx, t, db, requested)
	if err = Initialize(ctx, db, "mysql"); err == nil {
		t.Fatal("old layout accepted")
	}
	script, err := os.ReadFile("migrations/029-warmup-deadline.mysql.sql")
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
	assertLegacyDeadlineRun(ctx, t, db, requested)
}

func seedLegacyDeadlineRun(ctx context.Context, t *testing.T, db *sql.DB, requested time.Time) {
	t.Helper()
	for _, statement := range []string{
		`INSERT INTO connectors(name,driver,owner_id,status,created_at,updated_at) VALUES('main','sqlite','owner','active',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO namespaces(owner_id,name,title,status,created_at,updated_at) VALUES('owner','general','General','active',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO components(id,slug,title,owner_id,status,default_connector_name,component_scope,component_name,created_at,updated_at) VALUES('reader','reader','Reader','owner','draft','main','example.com/reader','reader',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO component_versions(report_id,version_no,state,authoring_mode,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES('reader',1,'draft','dql','{}','1','hash','{}','pending','test','test',1,'owner',CURRENT_TIMESTAMP)`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO component_warmup_runs(run_id,report_id,version_no,source_revision,spec_hash,plan_key,active_key,status,requested_by,target_json,requested_at) VALUES('legacy','reader',1,1,'hash','plan','legacy','running','owner','{}',?)`, requested); err != nil {
		t.Fatal(err)
	}
}

func assertLegacyDeadlineRun(ctx context.Context, t *testing.T, db *sql.DB, requested time.Time) {
	t.Helper()
	var status string
	var got time.Time
	var deadline sql.NullTime
	if err := db.QueryRowContext(ctx, "SELECT status,requested_at,deadline_at FROM component_warmup_runs WHERE run_id='legacy'").Scan(&status, &got, &deadline); err != nil || status != "running" || !got.Equal(requested) || deadline.Valid {
		t.Fatalf("legacy job rewritten: status=%s requested=%v deadline=%v err=%v", status, got, deadline, err)
	}
}
