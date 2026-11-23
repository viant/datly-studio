package schema

import (
	"database/sql"
	"os"
	"strings"
	"testing"
)

func TestWarmupSchedulesMigration30(t *testing.T) {
	ctx := t.Context()
	db := openSQLite(t, "warmup_schedule_migration")
	defer db.Close()
	ddl, e := canonicalDDL("sqlite")
	if e != nil {
		t.Fatal(e)
	}
	a := strings.Index(ddl, "CREATE TABLE component_warmup_schedules (")
	b := strings.Index(ddl, "CREATE TABLE component_warmup_runs (")
	if a < 0 || b < a {
		t.Fatal("schedule table not in canonical graph")
	}
	old := strings.Replace(ddl[:a]+ddl[b:], "VALUES (30)", "VALUES (29)", 1)
	if _, e = db.ExecContext(ctx, old); e != nil {
		t.Fatal(e)
	}
	if e = Initialize(ctx, db, "sqlite"); e == nil {
		t.Fatal("startup accepted a missing schedule table")
	}
	var count int
	if e = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE name='component_warmup_schedules'").Scan(&count); e != nil || count != 0 {
		t.Fatal("startup changed schema", e)
	}
	script, e := os.ReadFile("migrations/030-warmup-schedules.sqlite.sql")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.ExecContext(ctx, string(script)); e != nil {
		t.Fatal(e)
	}
	if e = Verify(ctx, db, "sqlite"); e != nil {
		t.Fatal(e)
	}
	v, e := SQLiteVersion(ctx, db)
	if e != nil || v != 30 {
		t.Fatalf("version=%d err=%v", v, e)
	}
	mysql, e := os.ReadFile("migrations/030-warmup-schedules.mysql.sql")
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(string(mysql), "ALTER TABLE component_warmup_runs") || strings.Contains(string(mysql), "fk_component_warmup_schedules_namespace") {
		t.Fatal("migration must not change run history or reference a nonunique namespace key")
	}
}

// This test requires a separate empty, exclusively owned MySQL database.
func TestWarmupSchedulesMigration30MySQL(t *testing.T) {
	dsn := os.Getenv("STUDIO_SCHEDULE_MIGRATION_MYSQL_DSN")
	if dsn == "" {
		t.Skip("owned migration MySQL fixture required")
	}
	ctx := t.Context()
	db, e := sql.Open("mysql", dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	var count int
	if e = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE()").Scan(&count); e != nil || count != 0 {
		t.Fatal("requires empty owned database", e)
	}
	ddl, e := MySQLDDL()
	if e != nil {
		t.Fatal(e)
	}
	a := strings.Index(ddl, "CREATE TABLE component_warmup_schedules (")
	b := strings.Index(ddl, "CREATE TABLE component_warmup_runs (")
	old := strings.Replace(ddl[:a]+ddl[b:], "VALUES (30)", "VALUES (29)", 1)
	for _, q := range mysqlStatements(old) {
		if _, e = db.ExecContext(ctx, q); e != nil {
			t.Fatal(e)
		}
	}
	if e = Initialize(ctx, db, "mysql"); e == nil {
		t.Fatal("startup accepted missing schedule table")
	}
	script, e := os.ReadFile("migrations/030-warmup-schedules.mysql.sql")
	if e != nil {
		t.Fatal(e)
	}
	for _, q := range mysqlStatements(string(script)) {
		if _, e = db.ExecContext(ctx, q); e != nil {
			t.Fatal(e)
		}
	}
	if e = Verify(ctx, db, "mysql"); e != nil {
		t.Fatal(e)
	}
	var version int
	if e = db.QueryRowContext(ctx, "SELECT version FROM schema_version").Scan(&version); e != nil || version != 30 {
		t.Fatalf("version=%d err=%v", version, e)
	}
}
