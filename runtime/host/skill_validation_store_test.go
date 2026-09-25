package host

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	studiors "github.com/viant/datly-studio/runtime/resources"
	_ "modernc.org/sqlite"
)

func TestValidateSkillToolReferencesExactStoredIdentity(t *testing.T) {
	db, err := sql.Open("sqlite", "file:skill-validation-identity?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE report_skill_roots(report_id TEXT,version_no INTEGER,skill_id TEXT,folder_id TEXT,skill_root TEXT)`,
		`CREATE TABLE report_resource_folders(report_id TEXT,version_no INTEGER,folder_id TEXT,namespace TEXT,root_path TEXT)`,
		`CREATE TABLE report_resource_files(report_id TEXT,version_no INTEGER,namespace TEXT,resource_path TEXT,content BLOB)`,
		`INSERT INTO report_skill_roots VALUES('report',1,'guide','folder','nested')`,
		`INSERT INTO report_skill_roots VALUES('report',2,'wrong-version','folder','nested')`,
		`INSERT INTO report_skill_roots VALUES('other',1,'wrong-report','folder','nested')`,
		`INSERT INTO report_resource_folders VALUES('report',1,'folder','guide.docs','manual')`,
		`INSERT INTO report_resource_folders VALUES('report',2,'folder','guide.docs','manual')`,
		`INSERT INTO report_resource_folders VALUES('other',1,'folder','guide.docs','manual')`,
		`INSERT INTO report_resource_files VALUES('report',1,'guide.docs','manual/nested/SKILL.md','---
name: guide
description: Guide
allowed-tools: skills/list
---
Use it.')`,
		`INSERT INTO report_resource_files VALUES('report',1,'other.namespace','manual/nested/SKILL.md','---
allowed-tools: missing.tool
---')`,
		`INSERT INTO report_resource_files VALUES('report',1,'guide.docs','manual/SKILL.md','---
allowed-tools: missing.tool
---')`,
		`INSERT INTO report_resource_files VALUES('report',2,'guide.docs','manual/nested/SKILL.md','---
allowed-tools: missing.tool
---')`,
		`INSERT INTO report_resource_files VALUES('other',1,'guide.docs','manual/nested/SKILL.md','---
allowed-tools: missing.tool
---')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	versions := []studiors.Version{{ReportID: "report", VersionNo: 1}}
	if err := validateSkillToolReferences(context.Background(), db, versions, nil); err != nil {
		t.Fatalf("exact identity: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM report_resource_files WHERE report_id='report' AND version_no=1 AND namespace='guide.docs' AND resource_path='manual/nested/SKILL.md'`); err != nil {
		t.Fatal(err)
	}
	err = validateSkillToolReferences(context.Background(), db, versions, nil)
	if !errors.Is(err, sql.ErrNoRows) || !strings.Contains(err.Error(), "skill guide content") {
		t.Fatalf("missing exact content error = %v", err)
	}
}
