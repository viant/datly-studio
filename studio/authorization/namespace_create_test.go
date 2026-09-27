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

func TestNewPrincipalCanCreateFirstNamespaceThroughProductionAuthorizer(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	authorizer, err := NewSDKAuthorizer(db)
	if err != nil {
		t.Fatal(err)
	}
	defer authorizer.Close(ctx)
	client, err := sdk.NewClient(&sqltransport.Transport{DB: db, Authorizer: authorizer})
	if err != nil {
		t.Fatal(err)
	}
	owner := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "first-user"})
	created, err := client.Namespaces().Create(owner, sdk.CreateNamespaceInput{Name: "scale", Title: "Scale Review"})
	if err != nil || created == nil || created.Name != "scale" || created.OwnerID != "first-user" {
		t.Fatalf("first namespace=%+v err=%v", created, err)
	}
	other := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "other-user"})
	if _, err = client.Namespaces().Get(other, "scale"); err == nil {
		t.Fatal("another principal read the new private namespace")
	}
}
