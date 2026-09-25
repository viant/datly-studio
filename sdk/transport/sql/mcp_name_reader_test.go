package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	_ "modernc.org/sqlite"
)

func TestMCPNameReaderSelectsOtherLiveLatestAndActiveRevisions(t *testing.T) {
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
	for _, id := range []string{"current", "other", "removed"} {
		if _, err := db.ExecContext(ctx, `INSERT INTO reports(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at)
			VALUES(?,?,?,?,?,'draft','main',?,?,1,?,?)`, id, "general", id, id, "owner", "reports/"+id, "reader", now, now); err != nil {
			t.Fatal(err)
		}
		count := 1
		if id != "current" {
			count = 3
		}
		for version := 1; version <= count; version++ {
			var generated any = fmt.Sprintf("SELECT %d", version)
			if id == "other" && version == 1 {
				generated = nil
			}
			if _, err := db.ExecContext(ctx, `INSERT INTO report_versions
				(report_id,version_no,state,authoring_mode,authored_dql,component_spec_json,spec_format_version,spec_hash,
				 type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at,generated_dql)
				 VALUES(?,?,'draft','dql',?,'{}','studio.v1',?,'{}','pending','v1','studio.v1',1,'owner',?,?)`,
				id, version, fmt.Sprintf("SELECT %d", version), fmt.Sprintf("%064d", version), now, generated); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO report_publications(report_id,active_version_no,desired_generation,publication_status,spec_hash,published_by,published_at)
		VALUES('other',1,1,'active',?,'owner',?)`, fmt.Sprintf("%064d", 1), now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE reports SET deleted_at=? WHERE id='removed'`, now); err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db}
	rows, err := transport.readMCPNameSources(ctx, "current")
	if err != nil || len(rows) != 2 {
		t.Fatalf("candidate rows=%+v err=%v", rows, err)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].VersionNo < rows[j].VersionNo })
	if rows[0].ReportId != "other" || rows[0].VersionNo != 1 || rows[0].GeneratedDql != nil ||
		rows[0].AuthoredDql == nil || *rows[0].AuthoredDql != "SELECT 1" ||
		rows[1].ReportId != "other" || rows[1].VersionNo != 3 || rows[1].GeneratedDql == nil || *rows[1].GeneratedDql != "SELECT 3" {
		t.Fatalf("latest/active candidates=%+v", rows)
	}
	if _, err := db.ExecContext(ctx, `UPDATE reports SET deleted_at=? WHERE id='other'`, now); err != nil {
		t.Fatal(err)
	}
	rows, err = transport.readMCPNameSources(ctx, "current")
	if err != nil || len(rows) != 0 {
		t.Fatalf("deleted report candidates=%+v err=%v", rows, err)
	}
}
