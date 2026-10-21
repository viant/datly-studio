package sqltransport

import (
	"context"
	"database/sql"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/runtime/accessprovider"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	_ "modernc.org/sqlite"
	"path/filepath"
	"testing"
	"time"
)

type reportCatalogFacts struct{ facts authz.Facts }

func (p reportCatalogFacts) Resolve(context.Context) (authz.Facts, error) { return p.facts, nil }

func TestReportCatalogBindsConnectorForNamespaceRoles(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(context.Background(), db, "studio"); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`INSERT INTO namespaces(namespace_id,owner_id,name,title,status,allowed_roles_json,etag,created_at,updated_at) VALUES('ns','owner','test','Test','active','{"view":[],"manage":["manager"]}',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO connectors(name,driver,owner_id,status,etag,created_at,updated_at) VALUES('main','sqlite','owner','active',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO components(id,slug,title,owner_id,status,namespace,namespace_id,default_connector_name,component_scope,component_name,etag,created_at,updated_at) VALUES('report','report','Report','owner','active','test','ns','main','test','report',1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	transport := &Transport{DB: db}
	for _, tc := range []struct {
		subject, role string
		allowed       bool
	}{{"owner", "reader", true}, {"delegate", "manager", true}, {"outsider", "reader", false}} {
		t.Run(tc.subject, func(t *testing.T) {
			facts := authz.Facts{Subject: tc.subject, Roles: []string{tc.role}, Tenant: "test", Issuer: "test", ValidUntil: time.Now().Add(time.Minute)}
			ctx := sdk.WithPrincipal(accessprovider.WithProvider(context.Background(), reportCatalogFacts{facts}), sdk.Principal{Subject: tc.subject})
			report, err := transport.getReportValue(ctx, "report")
			if tc.allowed && (err != nil || report == nil || report.ID != "report") {
				t.Fatalf("authorized namespace reader failed: %+v %v", report, err)
			}
			if !tc.allowed && err == nil {
				t.Fatal("namespace capability bypassed read authorization")
			}
		})
	}
}
