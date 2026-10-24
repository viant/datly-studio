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

func TestSDKReleaseGrantAddsViewAndExactRunOnly(t *testing.T) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(context.Background(), db, "studio"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO connectors(name,driver,owner_id,status,options_json,etag,created_at,updated_at) VALUES('main','sqlite','owner','active','{}',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO components(id,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES('reader','reader','Reader','owner','active','main','reports','reader',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	authorizer, err := NewSDKAuthorizer(db)
	if err != nil {
		t.Fatal(err)
	}
	defer authorizer.Close(context.Background())
	authorizer.ReleaseUse = func(ctx context.Context, request sqltransport.AuthorizationRequest) (bool, error) {
		p, _ := sdk.PrincipalFromContext(ctx)
		return p.Subject == "recipient" && request.ReportID == "reader" && request.Permission == "view" || p.Subject == "recipient" && request.ReportID == "reader" && request.Permission == "run" && request.VersionNo == "7", nil
	}
	recipient := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "recipient"})
	for _, request := range []sqltransport.AuthorizationRequest{{ReportID: "reader", Permission: "view"}, {ReportID: "reader", VersionNo: "7", Permission: "run"}} {
		if err = authorizer.Authorize(recipient, request); err != nil {
			t.Fatalf("released view/run denied %+v: %v", request, err)
		}
	}
	for _, request := range []sqltransport.AuthorizationRequest{{ReportID: "reader", VersionNo: "8", Permission: "run"}, {ReportID: "reader", Permission: "edit"}, {ReportID: "reader", Permission: "publish"}, {ReportID: "reader", Permission: "dql"}, {ReportID: "unrelated", Permission: "view"}} {
		if err = authorizer.Authorize(recipient, request); err == nil {
			t.Fatalf("release broadened %+v", request)
		}
	}
	unassigned := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "unassigned"})
	if err = authorizer.Authorize(unassigned, sqltransport.AuthorizationRequest{ReportID: "reader", VersionNo: "7", Permission: "run"}); err == nil {
		t.Fatal("unassigned actor ran released component")
	}
	if _, err = db.Exec(`UPDATE components SET deleted_at=CURRENT_TIMESTAMP WHERE id='reader'`); err != nil {
		t.Fatal(err)
	}
	if err = authorizer.Authorize(recipient, sqltransport.AuthorizationRequest{ReportID: "reader", VersionNo: "7", Permission: "run"}); err == nil {
		t.Fatal("deleted component revived by release grant")
	}
}
