package preview

import (
	"context"
	"reflect"
	"testing"

	"github.com/viant/datly-studio/internal/datatest"
)

func TestConnectorReaderKeepsPrincipalVisibilityAndPublicMode(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "preview_connectors", "studio")
	_, err := db.ExecContext(ctx, `
INSERT INTO connectors(name,driver,dsn_template,owner_id,status,options_json,etag,created_at,updated_at)
VALUES('owned','sqlite','file:owned.db','alice','active','{}',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO connectors(name,driver,dsn_template,owner_id,status,options_json,etag,created_at,updated_at)
VALUES('shared','sqlite','file:shared.db','bob','active','{}',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO connectors(name,driver,dsn_template,owner_id,status,options_json,etag,created_at,updated_at)
VALUES('hidden','sqlite','file:hidden.db','bob','active','{}',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO connectors(name,driver,dsn_template,owner_id,status,options_json,etag,created_at,updated_at)
VALUES('disabled','sqlite','file:disabled.db','alice','disabled','{}',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO connectors(name,driver,dsn_template,owner_id,status,options_json,etag,created_at,updated_at,deleted_at)
VALUES('deleted','sqlite','file:deleted.db','alice','deleted','{}',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at)
VALUES('bob','general','General','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO reports(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at)
VALUES('shared-report','general','shared-report','Shared','bob','active','shared','example.com/shared','reader',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO reports(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at)
VALUES('hidden-report','general','hidden-report','Hidden','bob','active','hidden','example.com/hidden','reader',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO report_acl(report_id,subject_type,subject_id,can_view,can_run)
VALUES('shared-report','user','alice',TRUE,FALSE);
INSERT INTO report_acl(report_id,subject_type,subject_id,can_view,can_run)
VALUES('hidden-report','role','alice',TRUE,FALSE);`)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := NewConnectorReader(db)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close(ctx)

	for _, test := range []struct {
		subject string
		want    []string
	}{
		{subject: "alice", want: []string{"owned", "shared"}},
		{subject: "bob", want: []string{"hidden", "shared"}},
		{subject: "unknown", want: nil},
	} {
		rows, readErr := reader.ListForPrincipal(ctx, test.subject)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if names := connectorNames(rows); !reflect.DeepEqual(names, test.want) {
			t.Fatalf("subject %s connectors=%v want %v", test.subject, names, test.want)
		}
	}
	all, err := reader.ListAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if names := connectorNames(all); !reflect.DeepEqual(names, []string{"hidden", "owned", "shared"}) {
		t.Fatalf("public connector set=%v", names)
	}
	if _, err := reader.ListForPrincipal(ctx, ""); err == nil {
		t.Fatal("empty subject selected connectors")
	}
	if _, err := db.ExecContext(ctx, `UPDATE reports SET deleted_at=CURRENT_TIMESTAMP WHERE id='shared-report'`); err != nil {
		t.Fatal(err)
	}
	afterDelete, err := reader.ListForPrincipal(ctx, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if names := connectorNames(afterDelete); !reflect.DeepEqual(names, []string{"owned"}) {
		t.Fatalf("deleted report still granted a connector: %v", names)
	}
}

func connectorNames(rows []connectorDefinition) []string {
	if len(rows) == 0 {
		return nil
	}
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row.Name)
	}
	return names
}
