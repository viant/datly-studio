package catalog

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/store/sql/fixture"
	_ "modernc.org/sqlite"
)

func TestComponentCatalogLoadsCanonicalPublishedReport(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatal(err)
	}
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	seed, err := fixture.SeedSampleCatalog(ctx, db, time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}

	components, err := NewComponentCatalog(db).LoadPublishedComponents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(components) != 1 {
		t.Fatalf("published reports = %d, want 1", len(components))
	}
	got := components[0]
	if got.Report.ID != seed.PublishedComponentID || got.Version.VersionNo != 1 || got.Version.AuthoringMode != "dql" {
		t.Fatalf("unexpected canonical published report: %+v", got)
	}
	if got.Publication.RuntimeRevision != "rev-0001" {
		t.Fatalf("runtime revision = %q, want rev-0001", got.Publication.RuntimeRevision)
	}
}

func TestComponentCatalogReadsAllLiveActiveExactVersions(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+fmt.Sprintf("%s/catalog.db", t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	for _, statement := range []string{
		`INSERT INTO connectors(name,driver,owner_id,status,etag,created_at,updated_at) VALUES('main','sqlite','owner','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at) VALUES('owner','general','General','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO runtime_generations(generation_no,source_revision,status,report_count,build_manifest_json,requested_by,requested_at,activated_at) VALUES(1,'runtime-1','active',135,'{}','owner',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	for i := range 137 {
		id := fmt.Sprintf("r-%03d", i)
		publishedAt := base.Add(time.Duration(i/2) * time.Minute)
		var deletedAt any
		if i == 0 {
			deletedAt = publishedAt
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO reports(id,namespace,slug,title,description,owner_id,status,default_connector_name,component_scope,component_name,current_draft_version,etag,created_at,updated_at,deleted_at)
			VALUES(?,'general',?,?,?,'owner','active','main','reports',?,2,2,?,?,?)`, id, id, id, "description-"+id, id, base, publishedAt, deletedAt); err != nil {
			t.Fatal(err)
		}
		for versionNo := 1; versionNo <= 2; versionNo++ {
			state := "superseded"
			if versionNo == 2 {
				state = "published"
			}
			if _, err := db.ExecContext(ctx, `INSERT INTO report_versions(report_id,version_no,state,authoring_mode,authored_dql,component_spec_json,spec_format_version,spec_hash,generated_dql,dql_export_limits_json,type_manifest_json,resource_manifest_json,component_descriptor_json,compile_status,compile_diagnostics_json,datly_version,compiler_version,source_revision,notes,created_by,created_at,validated_at,published_at)
				VALUES(?,? ,?,'dql',?, '{"views":[]}', 'v1', ?, ?, '{"max":100}', '{}', '{"files":[]}', '{"name":"reader"}', 'valid', '[]', 'v1', 'v1', ?, 'note','owner',?,?,?)`,
				id, versionNo, state, fmt.Sprintf("SELECT %d", versionNo), fmt.Sprintf("%064x", versionNo), fmt.Sprintf("SELECT %d", versionNo), versionNo, base, base, publishedAt); err != nil {
				t.Fatal(err)
			}
		}
		status := "active"
		if i == 1 {
			status = "pending"
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO report_publications(report_id,active_version_no,desired_version_no,desired_generation,active_generation,publication_status,runtime_revision,spec_hash,published_by,published_at,activated_at)
			VALUES(?,2,2,1,1,?,'runtime-1',?,'owner',?,?)`, id, status, fmt.Sprintf("%064x", 2), publishedAt, publishedAt); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, `UPDATE reports SET description=NULL,current_draft_version=NULL WHERE id='r-136'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE report_versions SET dql_export_limits_json=NULL,resource_manifest_json=NULL,component_descriptor_json=NULL,compile_diagnostics_json=NULL,notes=NULL WHERE report_id='r-136' AND version_no=2`); err != nil {
		t.Fatal(err)
	}
	components, err := NewComponentCatalog(db).LoadPublishedComponents(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(components) != 135 {
		t.Fatalf("live active component count=%d, want 135", len(components))
	}
	for index, item := range components {
		if item.Report == nil || item.Version == nil || item.Publication == nil || item.Version.VersionNo != 2 ||
			item.Version.AuthoredDQL != "SELECT 2" || item.Version.SourceRevision != 2 || item.Publication.ActiveGeneration == nil ||
			*item.Publication.ActiveGeneration != 1 || item.Publication.PublishedAt == nil || item.Report.OwnerPackage == "" {
			t.Fatalf("component %d has wrong active lineage: %+v", index, item)
		}
		if string(item.Version.ComponentSpec) != `{"views":[]}` {
			t.Fatalf("component %d lost JSON metadata: %+v", index, item.Version)
		}
		if item.Report.ID == "r-136" {
			if item.Report.Description != "" || item.Report.CurrentDraftVersion != nil || item.Version.DQLExportLimits != nil ||
				item.Version.ResourceManifest != nil || item.Version.ComponentDescriptor != nil || item.Version.CompileDiagnostics != nil || item.Version.Notes != "" {
				t.Fatalf("nullable catalog projection changed: %+v", item)
			}
		} else if item.Report.CurrentDraftVersion == nil || *item.Report.CurrentDraftVersion != 2 ||
			string(item.Version.ResourceManifest) != `{"files":[]}` || string(item.Version.ComponentDescriptor) != `{"name":"reader"}` {
			t.Fatalf("component %d lost draft pointer or JSON metadata: %+v", index, item)
		}
		if index > 0 {
			previous := components[index-1]
			if previous.Publication.PublishedAt.Before(*item.Publication.PublishedAt) ||
				previous.Publication.PublishedAt.Equal(*item.Publication.PublishedAt) && previous.Report.ID >= item.Report.ID {
				t.Fatalf("runtime catalog order %s before %s is invalid", previous.Report.ID, item.Report.ID)
			}
		}
	}
}
