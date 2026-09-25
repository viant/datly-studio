package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	_ "modernc.org/sqlite"
)

func TestVersionCatalogReaderKeepsRevisionAndJSONContract(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := db.ExecContext(ctx, `INSERT INTO connectors(name,driver,owner_id,status,options_json,etag,created_at,updated_at)
		VALUES('main','sqlite','owner','active','{}',1,?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at)
		VALUES('owner','general','General','active',1,?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO reports(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at)
		VALUES('report','general','report','Report','owner','draft','main','reports','reader',1,?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	const count = 503
	for version := 1; version <= count; version++ {
		state, status, actor := "draft", "pending", "owner"
		if version == count {
			state, status, actor = "validated", "valid", "reviewer"
		}
		var diagnostics any
		if version == count {
			diagnostics = `[{"message":"valid"}]`
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO report_versions
			(report_id,version_no,state,authoring_mode,authored_dql,component_spec_json,
			 spec_format_version,spec_hash,type_manifest_json,compile_status,
			 compile_diagnostics_json,datly_version,compiler_version,source_revision,created_by,created_at)
			 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			"report", version, state, "dql", "SELECT 1", fmt.Sprintf(`{"version":%d}`, version),
			"studio.v1", fmt.Sprintf("%064d", version), `{}`, status,
			diagnostics, "v1", "studio.v1", 1, actor, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db}
	first, err := transport.readVersionCatalog(ctx, versionCatalogRequest{ReportID: "report", Limit: 500})
	if err != nil || len(first) != 500 || first[0].VersionNo != count || first[499].VersionNo != 4 {
		t.Fatalf("first version page count=%d err=%v", len(first), err)
	}
	last, err := transport.readVersionCatalog(ctx, versionCatalogRequest{ReportID: "report", Limit: 500, Offset: 500})
	if err != nil || len(last) != 3 || last[0].VersionNo != 3 || last[2].VersionNo != 1 {
		t.Fatalf("tail version page=%+v err=%v", last, err)
	}
	latest, err := transport.getVersionValue(ctx, "report", count)
	if err != nil || latest.State != "validated" || latest.CompileStatus != "valid" || latest.CreatedBy != "reviewer" ||
		string(latest.ComponentSpec) != `{"version":503}` || string(latest.CompileDiagnostics) != `[{"message":"valid"}]` ||
		latest.ResourceManifest != nil || latest.ValidatedAt != nil {
		t.Fatalf("latest version=%+v err=%v", latest, err)
	}
	filtered, err := transport.readVersionCatalog(ctx, versionCatalogRequest{ReportID: "report", State: "validated",
		CompileStatus: "valid", CreatedBy: "reviewer", AuthoringMode: "dql", Limit: 10})
	if err != nil || len(filtered) != 1 || filtered[0].VersionNo != count {
		t.Fatalf("filtered versions=%+v err=%v", filtered, err)
	}
}
