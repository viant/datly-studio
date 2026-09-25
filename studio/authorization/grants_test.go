package authorization

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/viant/datly-studio/schema"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/scy/auth/jwt"
	xresponse "github.com/viant/xdatly/response"
	_ "modernc.org/sqlite"
)

func TestLifecycleGrantReaderRequiresLiveExactUserGrant(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`INSERT INTO connectors(name,driver,owner_id,status,etag,created_at,updated_at) VALUES('main','sqlite','owner','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO reports(id,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES('report','report','Report','owner','active','main','reports','report',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO report_acl(report_id,subject_type,subject_id,can_view,can_edit,can_publish) VALUES('report','user','alice',TRUE,FALSE,FALSE)`,
		`INSERT INTO report_acl(report_id,subject_type,subject_id,can_publish) VALUES('report','role','alice',TRUE)`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	provider := &dsql.SQLComponent{DB: db}
	if err := provider.RegisterConnector("studio", db); err != nil {
		t.Fatal(err)
	}
	claims := func(subject string) *jwt.Claims {
		value := &jwt.Claims{}
		value.Subject = subject
		return value
	}
	denied := func(err error) {
		t.Helper()
		var actual *xresponse.Error
		if !errors.As(err, &actual) || actual.Code != 403 {
			t.Fatalf("authorization error=%v, want 403", err)
		}
	}
	for _, permission := range []string{permissionView, permissionEdit, permissionPublish} {
		if err := AuthorizeReport(ctx, provider, claims("owner"), "report", permission); err != nil {
			t.Fatalf("owner %s: %v", permission, err)
		}
	}
	if err := AuthorizeReport(ctx, provider, claims("alice"), "report", permissionView); err != nil {
		t.Fatalf("user ACL view: %v", err)
	}
	denied(AuthorizeReport(ctx, provider, claims("alice"), "report", permissionEdit))
	denied(AuthorizeReport(ctx, provider, claims("alice"), "report", permissionPublish))
	denied(AuthorizeGlobal(ctx, provider, claims("alice"), permissionPublish))
	denied(AuthorizeGlobal(ctx, provider, claims("owner"), permissionPublish))
	denied(AuthorizeReport(ctx, provider, claims("alice"), "missing", permissionView))
	if _, err := db.ExecContext(ctx, `UPDATE report_acl SET can_publish=TRUE WHERE report_id='report' AND subject_type='user' AND subject_id='alice'`); err != nil {
		t.Fatal(err)
	}
	if err := AuthorizeReport(ctx, provider, claims("alice"), "report", permissionPublish); err != nil {
		t.Fatalf("user ACL publish: %v", err)
	}
	if err := AuthorizeGlobal(ctx, provider, claims("alice"), permissionPublish); err != nil {
		t.Fatalf("global user publish: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE reports SET deleted_at=CURRENT_TIMESTAMP WHERE id='report'`); err != nil {
		t.Fatal(err)
	}
	denied(AuthorizeReport(ctx, provider, claims("owner"), "report", permissionView))
	denied(AuthorizeReport(ctx, provider, claims("alice"), "report", permissionPublish))
	denied(AuthorizeGlobal(ctx, provider, claims("alice"), permissionPublish))
}
