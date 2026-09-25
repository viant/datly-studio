package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/viant/datly-studio/schema"
	_ "modernc.org/sqlite"
)

func TestServiceUpAndDown(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openTestDB(t)

	service, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := service.Up(ctx, db); err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	assertTableExists(t, ctx, db, "connectors")
	assertTableExists(t, ctx, db, "namespaces")
	assertTableExists(t, ctx, db, "reports")
	assertTableExists(t, ctx, db, "report_versions")
	assertTableExists(t, ctx, db, "report_fields")
	assertTableExists(t, ctx, db, "report_parameters")
	assertTableExists(t, ctx, db, "report_predicates")
	assertTableExists(t, ctx, db, "report_views")
	assertTableExists(t, ctx, db, "report_cube_configs")
	assertTableExists(t, ctx, db, "report_mcp_exposures")
	assertTableExists(t, ctx, db, "runtime_generations")
	assertTableExists(t, ctx, db, "report_warmup_runs")
	assertTableExists(t, ctx, db, "bff_sessions")
	assertTableExists(t, ctx, db, "report_publications")
	assertTableExists(t, ctx, db, "report_publication_events")
	assertTableExists(t, ctx, db, "report_acl")
	assertTableExists(t, ctx, db, "resource_namespace_claims")
	assertReportsConnectorFK(t, ctx, db)

	version, err := service.CurrentVersion(ctx, db)
	if err != nil {
		t.Fatalf("CurrentVersion() error = %v", err)
	}
	if version != schema.CanonicalVersion {
		t.Fatalf("CurrentVersion() = %d, want %d", version, schema.CanonicalVersion)
	}

	if err := service.Up(ctx, db); err != nil {
		t.Fatalf("second Up() error = %v", err)
	}

	if err := service.DownTo(ctx, db, 0); err != nil {
		t.Fatalf("DownTo() error = %v", err)
	}

	assertTableMissing(t, ctx, db, "connectors")
	assertTableMissing(t, ctx, db, "reports")

	version, err = service.CurrentVersion(ctx, db)
	if err != nil {
		t.Fatalf("CurrentVersion() after down error = %v", err)
	}
	if version != 0 {
		t.Fatalf("CurrentVersion() after down = %d, want 0", version)
	}
}

func TestServiceUpBackfillsResourceNamespaceClaims(t *testing.T) {
	for _, test := range []struct {
		name     string
		conflict bool
	}{{name: "one owner across file and folder"}, {name: "conflicting reports fail before table creation", conflict: true}} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			db := openTestDB(t)
			if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
				t.Fatal(err)
			}
			if _, err := db.ExecContext(ctx, `DROP TABLE resource_namespace_claims`); err != nil {
				t.Fatal(err)
			}
			if err := schema.SetSQLiteVersion(ctx, db, 12); err != nil {
				t.Fatal(err)
			}
			for _, statement := range []string{
				`INSERT INTO connectors(name,driver,owner_id,status,etag,created_at,updated_at) VALUES('main','sqlite','owner','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
				`INSERT INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at) VALUES('owner','general','General','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
				`INSERT INTO reports(id,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES('r1','r1','R1','owner','draft','main','reports','r1',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
				`INSERT INTO report_versions(report_id,version_no,state,authoring_mode,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES('r1',1,'draft','dql','{}','1','hash-r1','{}','pending','v1','v1',1,'owner',CURRENT_TIMESTAMP)`,
				`INSERT INTO report_resource_files(report_id,version_no,resource_id,namespace,resource_path,content,content_size,content_sha256,is_binary,created_at) VALUES('r1',1,'file-1','owner.docs','guide/SKILL.md','x',1,'digest',FALSE,CURRENT_TIMESTAMP)`,
				`INSERT INTO report_resource_folders(report_id,version_no,folder_id,namespace,root_path,uri_prefix) VALUES('r1',1,'folder-1','owner.docs','guide','skill://owner-guide/')`,
			} {
				if _, err := db.ExecContext(ctx, statement); err != nil {
					t.Fatal(err)
				}
			}
			if test.conflict {
				for _, statement := range []string{
					`INSERT INTO reports(id,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES('r2','r2','R2','owner','draft','main','reports','r2',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
					`INSERT INTO report_versions(report_id,version_no,state,authoring_mode,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES('r2',1,'draft','dql','{}','1','hash-r2','{}','pending','v1','v1',1,'owner',CURRENT_TIMESTAMP)`,
					`INSERT INTO report_resource_files(report_id,version_no,resource_id,namespace,resource_path,content,content_size,content_sha256,is_binary,created_at) VALUES('r2',1,'file-2','owner.docs','other.txt','y',1,'digest',FALSE,CURRENT_TIMESTAMP)`,
				} {
					if _, err := db.ExecContext(ctx, statement); err != nil {
						t.Fatal(err)
					}
				}
			}
			service, _ := New()
			err := service.Up(ctx, db)
			if test.conflict {
				if err == nil {
					t.Fatal("conflicting namespace migration succeeded")
				}
				version, versionErr := service.CurrentVersion(ctx, db)
				if versionErr != nil || version != 12 {
					t.Fatalf("failed migration version=%d err=%v", version, versionErr)
				}
				assertTableMissing(t, ctx, db, "resource_namespace_claims")
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var reportID, createdBy, updatedBy string
			if err := db.QueryRowContext(ctx, `SELECT report_id,created_by,updated_by FROM resource_namespace_claims WHERE namespace='owner.docs'`).Scan(&reportID, &createdBy, &updatedBy); err != nil || reportID != "r1" || createdBy != "system:migration" || updatedBy != "system:migration" {
				t.Fatalf("claim backfill report=%q actor=%q/%q err=%v", reportID, createdBy, updatedBy, err)
			}
			version, err := service.CurrentVersion(ctx, db)
			if err != nil || version != schema.CanonicalVersion {
				t.Fatalf("backfilled migration version=%d err=%v", version, err)
			}
		})
	}
}

func TestServiceUpAddsNamespaceFromCanonicalDDL(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	if _, err := db.ExecContext(ctx, `
CREATE TABLE reports (
  id VARCHAR(64) NOT NULL PRIMARY KEY,
  owner_id VARCHAR(128) NOT NULL,
  updated_at DATETIME NOT NULL
);
CREATE TABLE schema_version(version INTEGER NOT NULL);
INSERT INTO schema_version(version) VALUES (1);
INSERT INTO reports(id, owner_id, updated_at) VALUES ('legacy', 'alice', CURRENT_TIMESTAMP);`); err != nil {
		t.Fatal(err)
	}
	service, _ := New()
	if err := service.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	var namespace string
	if err := db.QueryRowContext(ctx, `SELECT namespace FROM reports WHERE id='legacy'`).Scan(&namespace); err != nil {
		t.Fatal(err)
	}
	if namespace != "general" {
		t.Fatalf("namespace=%q", namespace)
	}
	version, err := service.CurrentVersion(ctx, db)
	if err != nil || version != schema.CanonicalVersion {
		t.Fatalf("version=%d err=%v", version, err)
	}
	var count int
	if err = db.QueryRowContext(ctx, `SELECT COUNT(1) FROM namespaces WHERE owner_id='alice' AND name='general'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("namespace backfill count=%d err=%v", count, err)
	}
}

func TestServiceUpAddsDesiredPublicationVersion(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	if _, err := db.ExecContext(ctx, `
CREATE TABLE report_publications (
  report_id VARCHAR(64) NOT NULL PRIMARY KEY,
  active_version_no INT NOT NULL
);
CREATE TABLE schema_version(version INTEGER NOT NULL);
INSERT INTO schema_version(version) VALUES (3);
INSERT INTO report_publications(report_id,active_version_no) VALUES('reader',7);`); err != nil {
		t.Fatal(err)
	}
	service, _ := New()
	if err := service.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	var desired int
	if err := db.QueryRowContext(ctx, `SELECT desired_version_no FROM report_publications WHERE report_id='reader'`).Scan(&desired); err != nil || desired != 7 {
		t.Fatalf("desired version=%d err=%v", desired, err)
	}
	version, err := service.CurrentVersion(ctx, db)
	if err != nil || version != schema.CanonicalVersion {
		t.Fatalf("version=%d err=%v", version, err)
	}
	assertTableExists(t, ctx, db, "report_warmup_runs")
	assertTableExists(t, ctx, db, "bff_sessions")
}

func TestServiceUpAddsDurableBFFSessions(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	if _, err := db.ExecContext(ctx, `CREATE TABLE schema_version(version INTEGER NOT NULL); INSERT INTO schema_version(version) VALUES (5);`); err != nil {
		t.Fatal(err)
	}
	service, _ := New()
	if err := service.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	assertTableExists(t, ctx, db, "bff_sessions")
	version, err := service.CurrentVersion(ctx, db)
	if err != nil || version != schema.CanonicalVersion {
		t.Fatalf("version=%d err=%v", version, err)
	}
}

func TestServiceUpAddsOwnerScopedPublicationEvents(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	if _, err := db.ExecContext(ctx, `CREATE TABLE schema_version(version INTEGER NOT NULL); INSERT INTO schema_version(version) VALUES (6);`); err != nil {
		t.Fatal(err)
	}
	service, _ := New()
	if err := service.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	assertTableExists(t, ctx, db, "report_publication_events")
	version, err := service.CurrentVersion(ctx, db)
	if err != nil || version != schema.CanonicalVersion {
		t.Fatalf("version=%d err=%v", version, err)
	}
}

func TestServiceUpAddsReportACLEtag(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	if _, err := db.ExecContext(ctx, `
CREATE TABLE report_acl (
  report_id VARCHAR(64) NOT NULL,
  subject_type VARCHAR(32) NOT NULL,
  subject_id VARCHAR(128) NOT NULL,
  can_view BOOLEAN NOT NULL DEFAULT FALSE,
  can_run BOOLEAN NOT NULL DEFAULT FALSE,
  can_edit BOOLEAN NOT NULL DEFAULT FALSE,
  can_publish BOOLEAN NOT NULL DEFAULT FALSE,
  can_use_dql BOOLEAN NOT NULL DEFAULT FALSE,
  PRIMARY KEY (report_id, subject_type, subject_id)
);
CREATE TABLE schema_version(version INTEGER NOT NULL);
INSERT INTO schema_version(version) VALUES (7);
INSERT INTO report_acl(report_id,subject_type,subject_id,can_view) VALUES ('reader','user','alice',TRUE);`); err != nil {
		t.Fatal(err)
	}
	service, _ := New()
	if err := service.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	var etag int
	if err := db.QueryRowContext(ctx, `SELECT etag FROM report_acl WHERE report_id='reader' AND subject_id='alice'`).Scan(&etag); err != nil || etag != 1 {
		t.Fatalf("ACL etag=%d err=%v", etag, err)
	}
	version, err := service.CurrentVersion(ctx, db)
	if err != nil || version != schema.CanonicalVersion {
		t.Fatalf("version=%d err=%v", version, err)
	}
}

func TestServiceUpBackfillsWarmupAuditAndToken(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	if _, err := db.ExecContext(ctx, `
CREATE TABLE schema_version(version INTEGER NOT NULL);
INSERT INTO schema_version(version) VALUES (11);
CREATE TABLE report_warmup_runs (
  run_id TEXT PRIMARY KEY, status TEXT NOT NULL, requested_at DATETIME NOT NULL,
  requested_by TEXT NOT NULL, started_at DATETIME, completed_at DATETIME
);
INSERT INTO report_warmup_runs(run_id,status,requested_at,requested_by)
  VALUES('accepted','accepted','2026-09-24 10:00:00','alice');
INSERT INTO report_warmup_runs(run_id,status,requested_at,requested_by,started_at,completed_at)
  VALUES('completed','completed','2026-09-24 10:00:00','bob',
    '2026-09-24 10:01:00','2026-09-24 10:02:00');`); err != nil {
		t.Fatal(err)
	}
	service, _ := New()
	if err := service.Up(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		id, createdBy, updatedBy, updatedAt string
	}{
		{"accepted", "alice", "alice", "2026-09-24 10:00:00"},
		{"completed", "bob", "system:migration", "2026-09-24 10:02:00"},
	} {
		var createdAt, updatedAt, createdBy, updatedBy string
		if err := db.QueryRowContext(ctx, `SELECT created_at,updated_at,created_by,updated_by
			FROM report_warmup_runs WHERE run_id=?`, test.id).Scan(&createdAt, &updatedAt, &createdBy, &updatedBy); err != nil {
			t.Fatal(err)
		}
		if createdAt != "2026-09-24 10:00:00" || updatedAt != test.updatedAt ||
			createdBy != test.createdBy || updatedBy != test.updatedBy {
			t.Fatalf("%s audit created=%q/%q updated=%q/%q", test.id, createdAt, createdBy, updatedAt, updatedBy)
		}
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable foreign keys error = %v", err)
	}
	return db
}

func assertTableExists(t *testing.T, ctx context.Context, db *sql.DB, table string) {
	t.Helper()
	var name string
	err := db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
	if err != nil {
		t.Fatalf("table %s not found: %v", table, err)
	}
}

func assertTableMissing(t *testing.T, ctx context.Context, db *sql.DB, table string) {
	t.Helper()
	var name string
	err := db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
	if err == nil {
		t.Fatalf("table %s unexpectedly exists", table)
	}
	if err != sql.ErrNoRows {
		t.Fatalf("unexpected error checking table %s: %v", table, err)
	}
}

func assertReportsConnectorFK(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	rows, err := db.QueryContext(ctx, "PRAGMA foreign_key_list(reports)")
	if err != nil {
		t.Fatalf("foreign_key_list(reports) error = %v", err)
	}
	defer rows.Close()

	var (
		id       int
		seq      int
		table    string
		from     string
		to       string
		onUpdate string
		onDelete string
		match    string
		found    bool
	)
	for rows.Next() {
		if err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			t.Fatalf("scan foreign key row error = %v", err)
		}
		if table == "connectors" && from == "default_connector_name" && to == "name" {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate foreign_key_list(reports) error = %v", err)
	}
	if !found {
		t.Fatalf("reports.default_connector_name foreign key to connectors.name not found")
	}
}

func TestReportPublicationsRejectMissingVersion(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openTestDB(t)

	service, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := service.Up(ctx, db); err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	if _, err := db.ExecContext(ctx, `
	INSERT INTO connectors(name, driver, owner_id, status, etag, created_at, updated_at)
VALUES ('analytics', 'sqlite', 'system', 'active', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("insert connector error = %v", err)
	}
	if _, err := db.ExecContext(ctx, `
	INSERT INTO namespaces(owner_id, name, title, status, etag, created_at, updated_at)
VALUES ('system', 'general', 'General', 'active', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("insert namespace error = %v", err)
	}
	if _, err := db.ExecContext(ctx, `
	INSERT INTO reports(id, slug, title, owner_id, status, default_connector_name, component_scope, component_name, etag, created_at, updated_at)
VALUES ('r1', 'r1', 'Report 1', 'system', 'draft', 'analytics', 'reports', 'r1', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("insert report error = %v", err)
	}

	_, err = db.ExecContext(ctx, `
INSERT INTO report_publications(report_id, active_version_no, runtime_revision, published_by, published_at)
VALUES ('r1', 99, 'rev-1', 'system', CURRENT_TIMESTAMP)`)
	if err == nil {
		t.Fatalf("expected missing version foreign key violation")
	}
}
