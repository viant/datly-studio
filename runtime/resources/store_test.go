package resources

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	_ "modernc.org/sqlite"
)

func TestLoadBuildsVersionScopedStoreAndSkillFolders(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err = db.Exec(`INSERT INTO connectors(name,driver,owner_id,status,options_json,etag,created_at,updated_at) VALUES('main','sqlite','owner','active','{}',1,?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO reports(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES('report','general','report','Report','owner','draft','main','dynamic/owner/report','reader',1,?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO report_versions(report_id,version_no,state,authoring_mode,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES('report',1,'draft','dql','{}','studio.v1','hash','{}','pending','v1','studio',1,'owner',?)`, now); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct{ path, content string }{{"sql/records.sql", "SELECT 1"}, {"guide/SKILL.md", "---\nname: guide\ndescription: Test guide\n---\nUse [reference](references/guide.md)."}, {"guide/references/guide.md", "Reference"}} {
		if _, err = db.Exec(`INSERT INTO report_resource_files(report_id,version_no,resource_id,namespace,resource_path,content,content_size,content_sha256,is_binary,created_at) VALUES('report',1,?,'docs',?,?,?,'hash',0,?)`, item.path, item.path, item.content, len(item.content), now); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = db.Exec(`INSERT INTO report_resource_folders(report_id,version_no,folder_id,namespace,root_path,uri_prefix,ordinal) VALUES('report',1,'docs','docs','guide','skill://report-guide/',0)`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO report_skill_roots(report_id,version_no,skill_id,folder_id,skill_root,ordinal) VALUES('report',1,'guide','docs','.',0)`); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(ctx, db, []Version{{ReportID: "report", VersionNo: 1}})
	if err != nil {
		t.Fatal(err)
	}
	store := loaded.ByVersion[Version{ReportID: "report", VersionNo: 1}]
	if body, err := store.ReadFile("sql/records.sql"); err != nil || string(body) != "SELECT 1" {
		t.Fatalf("default resource=%q err=%v", body, err)
	}
	if len(loaded.Folders) != 1 || loaded.Folders[0].Namespace != "docs" || len(loaded.Folders[0].Skills) != 1 {
		t.Fatalf("folders=%+v", loaded.Folders)
	}
	if owner := loaded.ResourceReports["skill://report-guide/"]; owner != "report" {
		t.Fatalf("resource owner=%q", owner)
	}
}
