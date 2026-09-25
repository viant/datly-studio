package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	_ "modernc.org/sqlite"
)

func TestRuntimeReaderCatalogNestsChildrenAndScopesPrincipal(t *testing.T) {
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
		VALUES('main','sqlite','bob','active','{}',1,?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	for _, owner := range []string{"bob", "carol"} {
		if _, err := db.ExecContext(ctx, `INSERT INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at)
			VALUES(?,'general','General','active',1,?,?)`, owner, now, now); err != nil {
			t.Fatal(err)
		}
		id := "r-" + owner
		if _, err := db.ExecContext(ctx, `INSERT INTO reports(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at)
			VALUES(?,'general',?,?,?,'active','main',?,'reader',1,?,?)`, id, owner, owner, owner, "reports/"+owner, now, now); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO report_versions(report_id,version_no,state,authoring_mode,authored_dql,component_spec_json,
			spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at)
			VALUES(?,1,'published','dql','SELECT 1','{}','studio.v1',?,'{}','valid','v1','studio.v1',1,?,?)`,
			id, fmt.Sprintf("%064s", owner), owner, now); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO runtime_generations(generation_no,source_revision,status,report_count,build_manifest_json,requested_by,requested_at,activated_at)
		VALUES(1,'rev1','active',2,'{}','bob',?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	for _, owner := range []string{"bob", "carol"} {
		id := "r-" + owner
		if _, err := db.ExecContext(ctx, `INSERT INTO report_publications(report_id,active_version_no,desired_generation,active_generation,publication_status,runtime_revision,spec_hash,published_by,published_at,activated_at)
			VALUES(?,1,1,1,'active','rev1',?,?,?,?)`, id, fmt.Sprintf("%064s", owner), owner, now, now); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO report_acl(report_id,subject_type,subject_id,can_view,can_publish)
		VALUES('r-bob','user','alice',TRUE,TRUE)`); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 2; i++ {
		if _, err := db.ExecContext(ctx, `INSERT INTO report_mcp_exposures(report_id,version_no,exposure_id,route_id,route_method,route_path,kind,name,enabled,ordinal)
			VALUES('r-bob',1,?,?,'GET',?,'tool',?,TRUE,?)`, fmt.Sprintf("exposure-%d", i), fmt.Sprintf("route-%d", i), fmt.Sprintf("/route-%d", i), fmt.Sprintf("tool-%d", i), i); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO report_resource_folders(report_id,version_no,folder_id,namespace,root_path,uri_prefix,ordinal)
			VALUES('r-bob',1,?,'skills',?, ?, ?)`, fmt.Sprintf("folder-%d", i), fmt.Sprintf("guide-%d", i), fmt.Sprintf("skill://guide-%d/", i), i); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO report_skill_roots(report_id,version_no,skill_id,folder_id,skill_root,ordinal)
			VALUES('r-bob',1,?,? ,'.',?)`, fmt.Sprintf("skill-%d", i), fmt.Sprintf("folder-%d", i), i); err != nil {
			t.Fatal(err)
		}
	}
	transport := &Transport{DB: db}
	all, err := transport.readRuntimeReaderCatalog(ctx, 1)
	if err != nil || len(all) != 2 {
		t.Fatalf("all runtime readers=%+v err=%v", all, err)
	}
	var bob *sdk.RuntimeReader
	for i := range all {
		if all[i].ReportID == "r-bob" {
			bob = &all[i]
		}
	}
	if bob == nil || len(bob.MCPExposures) != 2 || len(bob.MCPResources) != 2 || len(bob.Skills) != 2 ||
		bob.MCPExposures[0].Name != "tool-1" || bob.MCPResources[1].URIPrefix != "skill://guide-2/" || bob.Skills[1].SkillID != "skill-2" {
		t.Fatalf("nested runtime reader=%+v", bob)
	}
	alice := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "alice"})
	visible, err := transport.readRuntimeReaderCatalog(alice, 1)
	if err != nil || len(visible) != 1 || visible[0].ReportID != "r-bob" {
		t.Fatalf("ACL-scoped readers=%+v err=%v", visible, err)
	}
	stranger := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "stranger"})
	hidden, err := transport.readRuntimeReaderCatalog(stranger, 1)
	if err != nil || len(hidden) != 0 {
		t.Fatalf("stranger readers=%+v err=%v", hidden, err)
	}
}
