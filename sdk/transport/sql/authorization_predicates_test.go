package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	_ "modernc.org/sqlite"
)

type dynamicPredicateFixture struct{}

func TestAuthorizationPredicateStoresDeclaredSQLScopeMetadata(t *testing.T) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(context.Background(), db, "studio"); err != nil {
		t.Fatal(err)
	}
	catalog, err := predicatecatalog.New(predicatecatalog.Package{Alias: "fixture", Path: "example.com/iam", Types: []reflect.Type{reflect.TypeFor[dynamicPredicateFixture]()}})
	if err != nil {
		t.Fatal(err)
	}
	client, err := sdk.NewClient(&Transport{DB: db, Authorizer: allowAuthorizer{}, Predicates: catalog})
	if err != nil {
		t.Fatal(err)
	}
	ctx := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "owner"})
	created, err := client.AuthorizationPredicates().Create(ctx, sdk.CreateAuthorizationPredicateInput{Name: "iam.vendor.read", Title: "Vendor access", PackagePath: "example.com/iam", TypeName: "dynamicPredicateFixture", Alias: "v", Columns: []string{"vendor_id", "tenant_id"}})
	if err != nil {
		t.Fatal(err)
	}
	if created.Alias != "v" || len(created.Columns) != 2 {
		t.Fatalf("metadata=%s %+v", created.Alias, created.Columns)
	}
	var raw string
	if err = db.QueryRow(`SELECT sql_scope_json FROM authorization_predicates WHERE name=?`, created.Name).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if raw != `{"alias":"v","columns":["vendor_id","tenant_id"]}` {
		t.Fatalf("sql_scope_json=%s", raw)
	}
}
