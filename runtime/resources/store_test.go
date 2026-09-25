package resources

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
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

type snapshotFixture struct {
	t  *testing.T
	db *sql.DB
}

func newSnapshotFixture(t *testing.T) *snapshotFixture {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := schema.ApplySQLite(context.Background(), db, "studio"); err != nil {
		t.Fatal(err)
	}
	f := &snapshotFixture{t: t, db: db}
	f.exec(`INSERT INTO connectors(name,driver,owner_id,status,options_json,etag,created_at,updated_at) VALUES('main','sqlite','owner','active','{}',1,?,?)`, time.Now().UTC(), time.Now().UTC())
	return f
}

func (f *snapshotFixture) exec(query string, args ...any) {
	f.t.Helper()
	if _, err := f.db.Exec(query, args...); err != nil {
		f.t.Fatal(err)
	}
}

func (f *snapshotFixture) version(reportID string, versionNo int) {
	f.t.Helper()
	now := time.Now().UTC()
	if versionNo == 1 {
		f.exec(`INSERT INTO reports(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES(?,'general',?,?,'owner','draft','main',?,'reader',1,?,?)`, reportID, reportID, reportID, "dynamic/owner/"+reportID, now, now)
	}
	f.exec(`INSERT INTO report_versions(report_id,version_no,state,authoring_mode,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) VALUES(?,?,'draft','dql','{}','studio.v1',?,'{}','pending','v1','studio',1,'owner',?)`, reportID, versionNo, fmt.Sprintf("hash-%d", versionNo), now)
}

func (f *snapshotFixture) file(reportID string, versionNo int, id, namespace, resourcePath, content string) {
	f.t.Helper()
	f.exec(`INSERT INTO report_resource_files(report_id,version_no,resource_id,namespace,resource_path,content,content_size,content_sha256,is_binary,created_at) VALUES(?,?,?,?,?,?,?,'hash',0,?)`, reportID, versionNo, id, namespace, resourcePath, []byte(content), len(content), time.Now().UTC())
}

func (f *snapshotFixture) folder(reportID string, versionNo int, id, namespace, root, uri string, ordinal int) {
	f.t.Helper()
	f.exec(`INSERT INTO report_resource_folders(report_id,version_no,folder_id,namespace,root_path,uri_prefix,ordinal) VALUES(?,?,?,?,?,?,?)`, reportID, versionNo, id, namespace, root, uri, ordinal)
}

func (f *snapshotFixture) skill(reportID string, versionNo int, id, folderID, root string, ordinal int) {
	f.t.Helper()
	f.exec(`INSERT INTO report_skill_roots(report_id,version_no,skill_id,folder_id,skill_root,ordinal) VALUES(?,?,?,?,?,?)`, reportID, versionNo, id, folderID, root, ordinal)
}

func TestLoadExactVersionAndMultipleVersions(t *testing.T) {
	f := newSnapshotFixture(t)
	f.version("report", 1)
	f.version("report", 2)
	f.version("other", 1)
	f.file("report", 1, "old", "docs", "old.txt", "old content")
	f.file("report", 2, "new", "docs", "new.txt", "new content")
	f.file("other", 1, "other", "other", "other.txt", "other content")
	f.folder("report", 1, "old-folder", "docs", ".", "resource://old/", 0)
	f.folder("report", 2, "late", "docs", ".", "resource://late/", 10)
	f.folder("report", 2, "early", "docs", ".", "resource://early/", 0)
	f.skill("report", 2, "z", "early", "z", 1)
	f.skill("report", 2, "b", "early", "b", 0)
	f.skill("report", 2, "a", "early", "a", 0)
	f.folder("other", 1, "other-folder", "other", ".", "resource://other/", 0)

	old := Version{ReportID: "report", VersionNo: 1}
	newVersion := Version{ReportID: "report", VersionNo: 2}
	onlyOld, err := Load(context.Background(), f.db, []Version{old})
	if err != nil {
		t.Fatal(err)
	}
	if len(onlyOld.ByVersion) != 1 || len(onlyOld.Folders) != 1 || onlyOld.ResourceReports["resource://early/"] != "" {
		t.Fatalf("exact version leaked: %+v", onlyOld)
	}
	if _, err := onlyOld.Store.ReadFile("docs:new.txt"); err == nil {
		t.Fatal("version 2 file leaked into version 1 snapshot")
	}
	loaded, err := Load(context.Background(), f.db, []Version{newVersion, old, {ReportID: "other", VersionNo: 1}})
	if err != nil {
		t.Fatal(err)
	}
	for version, resourcePath := range map[Version]struct{ path, body string }{
		old: {"old.txt", "old content"}, newVersion: {"new.txt", "new content"},
		{ReportID: "other", VersionNo: 1}: {"other.txt", "other content"},
	} {
		body, err := loaded.ByVersion[version].ReadFile(resourcePath.path)
		if err != nil || string(body) != resourcePath.body {
			t.Fatalf("%+v default %q: %q, %v", version, resourcePath.path, body, err)
		}
	}
	if _, err := loaded.ByVersion[old].ReadFile("new.txt"); err == nil {
		t.Fatal("version 2 default leaked into version 1")
	}
	if len(loaded.Folders) != 4 || loaded.Folders[0].URIPrefix != "resource://early/" || loaded.Folders[1].URIPrefix != "resource://late/" || loaded.Folders[2].URIPrefix != "resource://old/" || loaded.Folders[3].URIPrefix != "resource://other/" {
		t.Fatalf("folder order: %+v", loaded.Folders)
	}
	if got := strings.Join(loaded.Folders[0].Skills, ","); got != "a,b,z" {
		t.Fatalf("skill order: %s", got)
	}
	if loaded.ResourceReports["resource://early/"] != "report" || loaded.ResourceReports["resource://other/"] != "other" {
		t.Fatalf("folder owners: %+v", loaded.ResourceReports)
	}
	f.exec(`UPDATE report_resource_files SET content=? WHERE report_id='report' AND version_no=2`, []byte("changed"))
	if body, err := loaded.ByVersion[newVersion].ReadFile("new.txt"); err != nil || string(body) != "new content" {
		t.Fatalf("snapshot content changed: %q, %v", body, err)
	}
}

func TestLoadRejectsDuplicateResources(t *testing.T) {
	t.Run("default path in one version", func(t *testing.T) {
		f := newSnapshotFixture(t)
		f.version("report", 1)
		f.file("report", 1, "one", "docs", "same.txt", "one")
		f.file("report", 1, "two", "other", "same.txt", "two")
		if _, err := Load(context.Background(), f.db, []Version{{ReportID: "report", VersionNo: 1}}); err == nil || !strings.Contains(err.Error(), "duplicate default resource path") {
			t.Fatalf("expected duplicate default path, got %v", err)
		}
	})
	t.Run("namespace path across versions", func(t *testing.T) {
		f := newSnapshotFixture(t)
		f.version("report", 1)
		f.version("report", 2)
		f.file("report", 1, "one", "docs", "same.txt", "one")
		f.file("report", 2, "two", "docs", "same.txt", "two")
		if _, err := Load(context.Background(), f.db, []Version{{ReportID: "report", VersionNo: 1}, {ReportID: "report", VersionNo: 2}}); err == nil || !strings.Contains(err.Error(), "duplicate resource namespace/path") {
			t.Fatalf("expected namespace/path collision, got %v", err)
		}
	})
	t.Run("URI owner across reports", func(t *testing.T) {
		f := newSnapshotFixture(t)
		f.version("one", 1)
		f.version("two", 1)
		f.folder("one", 1, "a", "docs", ".", "resource://shared/", 0)
		f.folder("two", 1, "b", "other", ".", "resource://shared/", 0)
		if _, err := Load(context.Background(), f.db, []Version{{ReportID: "one", VersionNo: 1}, {ReportID: "two", VersionNo: 1}}); err == nil || !strings.Contains(err.Error(), "owned by both reports") {
			t.Fatalf("expected conflicting owner, got %v", err)
		}
	})
}

func TestLoadRejectsMalformedPaths(t *testing.T) {
	for _, resourcePath := range []string{".", "../escape", "a/../b", "/absolute", "a\\b", "a//b", " trailing.txt "} {
		t.Run(resourcePath, func(t *testing.T) {
			f := newSnapshotFixture(t)
			f.version("report", 1)
			f.file("report", 1, "bad", "docs", resourcePath, "content")
			if _, err := Load(context.Background(), f.db, []Version{{ReportID: "report", VersionNo: 1}}); err == nil || !strings.Contains(err.Error(), "invalid resource path") {
				t.Fatalf("expected invalid path, got %v", err)
			}
		})
	}
	t.Run("skill root", func(t *testing.T) {
		f := newSnapshotFixture(t)
		f.version("report", 1)
		f.folder("report", 1, "docs", "docs", ".", "resource://docs/", 0)
		f.skill("report", 1, "bad", "docs", "../escape", 0)
		if _, err := Load(context.Background(), f.db, []Version{{ReportID: "report", VersionNo: 1}}); err == nil || !strings.Contains(err.Error(), "invalid skill root") {
			t.Fatalf("expected invalid skill root, got %v", err)
		}
	})
}
