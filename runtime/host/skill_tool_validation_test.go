package host

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	studiors "github.com/viant/datly-studio/runtime/resources"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	_ "modernc.org/sqlite"
)

func TestValidateSkillToolReferencesRejectsUnknownAndDuplicateTools(t *testing.T) {
	db, err := sql.Open("sqlite", "file:skill-tool-validation?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, ddl := range []string{
		`CREATE TABLE report_skill_roots(report_id TEXT,version_no INTEGER,skill_id TEXT,folder_id TEXT,skill_root TEXT)`,
		`CREATE TABLE report_resource_folders(report_id TEXT,version_no INTEGER,folder_id TEXT,namespace TEXT,root_path TEXT)`,
		`CREATE TABLE report_resource_files(report_id TEXT,version_no INTEGER,namespace TEXT,resource_path TEXT,content BLOB)`,
	} {
		if _, err = db.Exec(ddl); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = db.Exec(`INSERT INTO report_skill_roots VALUES('report',1,'guide','folder','.')`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO report_resource_folders VALUES('report',1,'folder','guide.docs','guide')`); err != nil {
		t.Fatal(err)
	}
	component := &registry.RegisteredComponent{Component: &spec.Component{Key: spec.Key{Kind: spec.KindComponent, Name: "Reader"}, Routes: []*spec.Route{{Method: "GET", Path: "/records", MCP: []*spec.MCPExposure{{Kind: spec.MCPExposureTool, Name: "records.read"}}}}}}
	check := func(content string) error {
		if _, err = db.Exec(`DELETE FROM report_resource_files`); err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(`INSERT INTO report_resource_files VALUES('report',1,'guide.docs','guide/SKILL.md',?)`, content); err != nil {
			t.Fatal(err)
		}
		return validateSkillToolReferences(context.Background(), db, []studiors.Version{{ReportID: "report", VersionNo: 1}}, []*registry.RegisteredComponent{component})
	}
	if err = check("---\nname: guide\ndescription: Guide\nallowed-tools: records.read\n---\nUse it."); err != nil {
		t.Fatalf("valid tool list: %v", err)
	}
	if err = check("---\nname: guide\ndescription: Guide\nallowed-tools: skills/list skills/get\n---\nUse it."); err != nil {
		t.Fatalf("compatibility tool list: %v", err)
	}
	if err = check("---\nname: guide\ndescription: Guide\nallowed-tools: missing.tool\n---\nUse it."); err == nil || !strings.Contains(err.Error(), "absent from this generation") {
		t.Fatalf("unknown tool error=%v", err)
	}
	if err = check("---\nname: guide\ndescription: Guide\nallowed-tools: records.read records.read\n---\nUse it."); err == nil || !strings.Contains(err.Error(), "repeats tool") {
		t.Fatalf("duplicate tool error=%v", err)
	}
}
