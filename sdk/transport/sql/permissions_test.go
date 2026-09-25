package sqltransport

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	_ "modernc.org/sqlite"
)

func TestReportCapabilitiesUseExactDatlyReaderAndDirectUserACL(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys=ON`); err != nil {
		t.Fatal(err)
	}
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `
INSERT INTO connectors(name,driver,owner_id,status,etag,created_at,updated_at)
VALUES('main','sqlite','owner','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at)
VALUES('owner','general','General','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO reports(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at)
VALUES('report','general','report','Report','owner','active','main','example.com/report','reader',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
INSERT INTO report_acl(report_id,subject_type,subject_id,can_view,can_run,can_edit,can_publish,can_use_dql)
VALUES('report','user','viewer',TRUE,TRUE,FALSE,FALSE,FALSE);
INSERT INTO report_acl(report_id,subject_type,subject_id,can_view,can_run,can_edit,can_publish,can_use_dql)
VALUES('report','user','editor',TRUE,TRUE,TRUE,FALSE,FALSE);
INSERT INTO report_acl(report_id,subject_type,subject_id,can_view,can_run,can_edit,can_publish,can_use_dql)
VALUES('report','role','analyst',TRUE,TRUE,TRUE,TRUE,TRUE);`)
	if err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db}
	defer transport.Close(ctx)
	for _, test := range []struct {
		subject string
		want    sdk.ReportCapabilities
	}{
		{subject: "owner", want: sdk.ReportCapabilities{CanView: true, CanRun: true, CanEdit: true, CanPublish: true, CanUseDQL: true, CanManageACL: true}},
		{subject: "viewer", want: sdk.ReportCapabilities{CanView: true, CanRun: true}},
		{subject: "editor", want: sdk.ReportCapabilities{CanView: true, CanRun: true, CanEdit: true}},
		{subject: "analyst", want: sdk.ReportCapabilities{}},
	} {
		principal := sdk.WithPrincipal(ctx, sdk.Principal{Subject: test.subject})
		actual, readErr := transport.reportCapabilities(principal, "report")
		if readErr != nil || actual != test.want {
			t.Fatalf("subject %s capabilities=%+v err=%v want %+v", test.subject, actual, readErr, test.want)
		}
	}
	for _, subject := range []string{"owner", "editor", "viewer"} {
		result := struct {
			Items []*sdk.ReportACL `json:"items"`
		}{}
		listErr := transport.acl(sdk.WithPrincipal(ctx, sdk.Principal{Subject: subject}), sdk.OperationACLList,
			map[string]string{"reportId": "report"}, &result)
		if subject == "owner" {
			if listErr != nil || len(result.Items) != 3 {
				t.Fatalf("owner ACL list=%+v err=%v", result, listErr)
			}
			continue
		}
		var sdkErr *sdk.Error
		if !errors.As(listErr, &sdkErr) || sdkErr.Code != sdk.ErrorForbidden || len(result.Items) != 0 {
			t.Fatalf("%s ACL list=%+v err=%v, want forbidden", subject, result, listErr)
		}
	}
	entries, err := transport.readACLList(ctx, "report")
	if err != nil || len(entries) != 3 || entries[0].SubjectType != "role" || entries[0].SubjectID != "analyst" ||
		entries[1].SubjectType != "user" || entries[1].SubjectID != "editor" || entries[2].SubjectID != "viewer" || entries[2].ETag != 1 {
		t.Fatalf("ACL reader list=%+v err=%v", entries, err)
	}
	exact, err := transport.readACLOne(ctx, "report", "user", "viewer")
	if err != nil || exact.SubjectID != "viewer" || !exact.CanRun || exact.CanPublish {
		t.Fatalf("ACL reader identity=%+v err=%v", exact, err)
	}
	if _, err := transport.reportCapabilities(sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner"}), "missing"); err == nil {
		t.Fatal("missing report returned capabilities")
	}
	if _, err := db.ExecContext(ctx, `UPDATE reports SET deleted_at=CURRENT_TIMESTAMP WHERE id='report'`); err != nil {
		t.Fatal(err)
	}
	if _, err := transport.reportCapabilities(sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner"}), "report"); err == nil {
		t.Fatal("deleted report returned capabilities")
	}
}
