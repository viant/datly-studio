package authorization

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	sqltransport "github.com/viant/datly-studio/sdk/transport/sql"
	_ "modernc.org/sqlite"
)

func TestSDKAuthorizerEnforcesOwnerAndACL(t *testing.T) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(context.Background(), db, "studio"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err = db.Exec(`INSERT INTO connectors(name,driver,owner_id,status,options_json,etag,created_at,updated_at) VALUES ('main','sqlite','bob','active','{}',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at) VALUES ('bob','general','General','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO reports(id,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES ('shared','shared','Shared','bob','active','main','reports','shared',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO report_acl(report_id,subject_type,subject_id,can_view,can_run,can_edit,can_use_dql) VALUES ('shared','user','alice',TRUE,TRUE,TRUE,FALSE)`); err != nil {
		t.Fatal(err)
	}
	authorizer, err := NewSDKAuthorizer(db)
	if err != nil {
		t.Fatal(err)
	}
	if authorizer.ReportCapabilities == nil {
		t.Fatal("production authorizer did not retain its generated capability reader")
	}
	defer func() {
		if closeErr := authorizer.Close(context.Background()); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	alice := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "alice"})
	bob := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "bob"})
	if err = authorizer.Authorize(alice, sqltransport.AuthorizationRequest{ReportID: "shared", Permission: "edit"}); err != nil {
		t.Fatal(err)
	}
	if err = (SDKAuthorizer{DB: db}).Authorize(alice, sqltransport.AuthorizationRequest{ReportID: "shared", Permission: "edit"}); err != nil {
		t.Fatalf("short-lived compatibility reader: %v", err)
	}
	if err = authorizer.Authorize(alice, sqltransport.AuthorizationRequest{ReportID: "shared", Permission: "run"}); err != nil {
		t.Fatal(err)
	}
	if err = authorizer.Authorize(alice, sqltransport.AuthorizationRequest{NamespaceName: "general", Permission: "view"}); err != nil {
		t.Fatalf("namespace ACL view: %v", err)
	}
	if err = authorizer.Authorize(alice, sqltransport.AuthorizationRequest{NamespaceName: "general", Permission: "run"}); err != nil {
		t.Fatalf("namespace ACL run: %v", err)
	}
	if err = authorizer.Authorize(alice, sqltransport.AuthorizationRequest{NamespaceName: "general", Permission: "edit"}); err == nil {
		t.Fatal("namespace ACL edit bypassed owner-only rule")
	}
	if err = authorizer.Authorize(bob, sqltransport.AuthorizationRequest{NamespaceName: "general", Permission: "edit"}); err != nil {
		t.Fatalf("namespace owner edit: %v", err)
	}
	if err = authorizer.Authorize(alice, sqltransport.AuthorizationRequest{ReportID: "shared", Permission: "dql"}); err == nil {
		t.Fatal("DQL access without can_use_dql unexpectedly allowed")
	}
	if _, err = db.Exec(`UPDATE report_acl SET can_use_dql=TRUE WHERE report_id='shared' AND subject_id='alice'`); err != nil {
		t.Fatal(err)
	}
	if err = authorizer.Authorize(alice, sqltransport.AuthorizationRequest{ReportID: "shared", Permission: "dql"}); err != nil {
		t.Fatal(err)
	}
	if err = authorizer.Authorize(alice, sqltransport.AuthorizationRequest{NamespaceName: "general", Permission: "dql"}); err != nil {
		t.Fatalf("namespace ACL DQL after grant: %v", err)
	}
	if err = authorizer.Authorize(alice, sqltransport.AuthorizationRequest{ConnectorName: "main", Permission: "edit"}); err != nil {
		t.Fatal(err)
	}
	if err = authorizer.Authorize(alice, sqltransport.AuthorizationRequest{OwnerID: "bob", Permission: "edit"}); err == nil {
		t.Fatal("cross-owner creation allowed")
	}
	if err = authorizer.Authorize(alice, sqltransport.AuthorizationRequest{ReportID: "missing", Permission: "view"}); err == nil {
		t.Fatal("missing report allowed")
	}
	if err = authorizer.Authorize(bob, sqltransport.AuthorizationRequest{Permission: "publish"}); err != nil {
		t.Fatalf("owner global publish authorization: %v", err)
	}
	if err = authorizer.Authorize(alice, sqltransport.AuthorizationRequest{Permission: "publish"}); err == nil {
		t.Fatal("viewer global publish authorization unexpectedly allowed")
	}
	if _, err = db.Exec(`UPDATE reports SET deleted_at=CURRENT_TIMESTAMP WHERE id='shared'`); err != nil {
		t.Fatal(err)
	}
	if err = authorizer.Authorize(bob, sqltransport.AuthorizationRequest{ReportID: "shared", Permission: "view"}); err == nil {
		t.Fatal("deleted report owner authorization unexpectedly allowed")
	}
	if err = authorizer.Authorize(alice, sqltransport.AuthorizationRequest{ReportID: "shared", Permission: "edit"}); err == nil {
		t.Fatal("deleted report ACL authorization unexpectedly allowed")
	}
	if err = authorizer.Authorize(alice, sqltransport.AuthorizationRequest{NamespaceName: "general", Permission: "view"}); err == nil {
		t.Fatal("namespace inherited ACL from deleted report")
	}
	if _, err = db.Exec(`UPDATE namespaces SET deleted_at=CURRENT_TIMESTAMP WHERE owner_id='bob' AND name='general'`); err != nil {
		t.Fatal(err)
	}
	if err = authorizer.Authorize(bob, sqltransport.AuthorizationRequest{NamespaceName: "general", Permission: "edit"}); err == nil {
		t.Fatal("deleted namespace owner authorization unexpectedly allowed")
	}
}
