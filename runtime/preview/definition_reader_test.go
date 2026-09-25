package preview

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/viant/datly-studio/internal/datatest"
)

func TestDefinitionReaderPinsReportAndVersionAndDeniesDeletedReports(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "preview_definition", "studio")
	_, err := db.ExecContext(ctx, `
INSERT INTO connectors(name,driver,dsn_template,owner_id,status,options_json,etag,created_at,updated_at)
VALUES('main','sqlite','file:main.db','owner','active','{"busy_timeout_ms":5000}',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at)
VALUES('owner','general','General','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO reports(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at)
VALUES('report','general','report','Report','owner','draft','main','example.com/report','reader',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO report_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at)
VALUES('report',1,'draft','dql','SELECT 1','SELECT 1','{}','studio.v1','one','{}','valid','v1','v1',1,'owner',CURRENT_TIMESTAMP);
INSERT INTO report_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at)
VALUES('report',2,'validated','dql','SELECT 2','SELECT 2','{}','studio.v1','two','{}','valid','v1','v1',2,'owner',CURRENT_TIMESTAMP);`)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := NewDefinitionReader(db)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close(ctx)
	for _, version := range []int{1, 2} {
		row, readErr := reader.Get(ctx, "report", version)
		if readErr != nil || row.VersionNo != version || row.ReportId != "report" || row.SourceRevision != int64(version) || row.SpecHash != []string{"", "one", "two"}[version] || row.DsnTemplate == nil || *row.DsnTemplate != "file:main.db" {
			t.Fatalf("version %d: row=%+v err=%v", version, row, readErr)
		}
		if string(row.OptionsJson) != `{"busy_timeout_ms":5000}` {
			t.Fatalf("connector options=%s", row.OptionsJson)
		}
	}
	for _, key := range []struct {
		id      string
		version int
	}{{"other", 1}, {"report", 3}} {
		if _, err = reader.Get(ctx, key.id, key.version); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("missing %+v: %v", key, err)
		}
	}
	if _, err = db.ExecContext(ctx, `UPDATE reports SET deleted_at=CURRENT_TIMESTAMP WHERE id='report'`); err != nil {
		t.Fatal(err)
	}
	if _, err = reader.Get(ctx, "report", 2); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("deleted report remained visible: %v", err)
	}
}
