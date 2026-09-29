package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	"testing"
)

func TestSelectedNamespaceSeparatesResourcesForSameOwner(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	client, err := sdk.NewClient(&Transport{DB: db, Authorizer: allowAuthorizer{}})
	if err != nil {
		t.Fatal(err)
	}
	owner := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner"})
	a, err := client.Namespaces().Create(owner, sdk.CreateNamespaceInput{Name: "alpha", Title: "Alpha"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := client.Namespaces().Create(owner, sdk.CreateNamespaceInput{Name: "beta", Title: "Beta"})
	if err != nil {
		t.Fatal(err)
	}
	connector, err := client.Connectors().Create(owner, sdk.CreateConnectorInput{Name: "shared", Driver: "sqlite"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE connectors SET status='active' WHERE name='shared'`); err != nil {
		t.Fatal(err)
	}
	selectedA := sdk.WithNamespaceSelection(owner, a.NamespaceID)
	selectedB := sdk.WithNamespaceSelection(owner, b.NamespaceID)
	create := func(ctx context.Context, slug string) *sdk.Component {
		row, err := client.Components().Create(ctx, sdk.CreateComponentInput{Slug: slug, Title: slug, DefaultConnectorName: connector.Name})
		if err != nil {
			t.Fatal(err)
		}
		return row
	}
	componentA, componentB := create(selectedA, "alpha-reader"), create(selectedB, "beta-reader")
	if componentA.Namespace != "alpha" || componentB.Namespace != "beta" {
		t.Fatal("selected namespace did not own creation")
	}
	for _, expected := range []struct {
		component   *sdk.Component
		namespaceID string
	}{{componentA, a.NamespaceID}, {componentB, b.NamespaceID}} {
		var storedID string
		if err := db.QueryRow(`SELECT namespace_id FROM components WHERE id=?`, expected.component.ID).Scan(&storedID); err != nil || storedID != expected.namespaceID {
			t.Fatalf("component ownership=%q want=%q err=%v", storedID, expected.namespaceID, err)
		}
	}
	page, err := client.Components().List(selectedA, sdk.ListComponentsInput{})
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != componentA.ID {
		t.Fatalf("namespace catalog: %+v err=%v", page, err)
	}
	if _, err := client.Components().List(selectedA, sdk.ListComponentsInput{Namespace: "beta"}); err == nil {
		t.Fatal("caller filter bypassed namespace")
	}
	_ = create(selectedA, "alpha-second")
	page, err = client.Components().List(selectedA, sdk.ListComponentsInput{Limit: 1, Offset: 1})
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != componentA.ID {
		t.Fatalf("namespace pagination was applied after filtering: %+v err=%v", page, err)
	}
	if _, err := client.Components().Get(selectedA, componentB.ID); err == nil {
		t.Fatal("same owner read another namespace")
	}
	if _, err := client.Versions().Create(selectedA, componentB.ID, sdk.CreateVersionInput{}); err == nil {
		t.Fatal("same owner authored another namespace")
	}
	move := "beta"
	if _, err := client.Components().Update(selectedA, componentA.ID, sdk.UpdateComponentInput{Namespace: &move, ETag: componentA.ETag}); err == nil {
		t.Fatal("component moved outside selected namespace")
	}
	if _, err := client.Components().Create(selectedA, sdk.CreateComponentInput{Namespace: "beta", Slug: "forged", Title: "Forged", DefaultConnectorName: connector.Name}); err == nil {
		t.Fatal("component created outside selected namespace")
	}
	connectors, err := client.Connectors().List(selectedA, sdk.ListConnectorsInput{})
	if err != nil || len(connectors.Items) != 1 || connectors.Items[0].Name != "shared" {
		t.Fatalf("global connector unavailable: %+v err=%v", connectors, err)
	}
	stranger := sdk.WithNamespaceSelection(sdk.WithPrincipal(ctx, sdk.Principal{Subject: "stranger"}), a.NamespaceID)
	if _, err := client.Components().List(stranger, sdk.ListComponentsInput{}); err == nil {
		t.Fatal("selected ID granted private namespace")
	}
	if _, err := client.Components().List(sdk.WithNamespaceSelection(owner, ""), sdk.ListComponentsInput{}); err == nil {
		t.Fatal("empty explicit selection fell back to all namespaces")
	}
	if _, err := db.Exec(`UPDATE namespaces SET status='archived' WHERE namespace_id=?`, a.NamespaceID); err != nil {
		t.Fatal(err)
	}
	directory, err := client.Namespaces().List(selectedA, sdk.ListNamespacesInput{})
	if err != nil || len(directory.Items) != 2 {
		t.Fatalf("archived selection blocked namespace recovery: %+v err=%v", directory, err)
	}
	connectors, err = client.Connectors().List(selectedA, sdk.ListConnectorsInput{})
	if err != nil || len(connectors.Items) != 1 {
		t.Fatalf("archived selection blocked global connectors: %+v err=%v", connectors, err)
	}
	if _, err := client.Components().Get(selectedA, componentA.ID); err == nil {
		t.Fatal("archived selection remained usable")
	}
}
