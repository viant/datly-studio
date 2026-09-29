package sqltransport

import (
	"context"
	"database/sql"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	_ "modernc.org/sqlite"
	"path/filepath"
	"testing"
)

func TestResourceNamesAreIndependentAcrossSelectedNamespaces(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(context.Background(), db, "studio"); err != nil {
		t.Fatal(err)
	}
	client, err := sdk.NewClient(&Transport{DB: db, Authorizer: allowAuthorizer{}})
	if err != nil {
		t.Fatal(err)
	}
	owner := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "owner"})
	connector, err := client.Connectors().Create(owner, sdk.CreateConnectorInput{Name: "shared", Driver: "sqlite"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE connectors SET status='active' WHERE name='shared'`); err != nil {
		t.Fatal(err)
	}
	type authored struct {
		ctx       context.Context
		component *sdk.Component
		version   *sdk.ReportVersion
		fileID    string
		revision  int64
	}
	var items []authored
	for _, name := range []string{"alpha", "beta"} {
		workspace, err := client.Namespaces().Create(owner, sdk.CreateNamespaceInput{Name: name, Title: name})
		if err != nil {
			t.Fatal(err)
		}
		selected := sdk.WithNamespaceSelection(owner, workspace.NamespaceID)
		component, err := client.Components().Create(selected, sdk.CreateComponentInput{Slug: name + "-reader", Title: name, DefaultConnectorName: connector.Name})
		if err != nil {
			t.Fatal(err)
		}
		version, err := client.Versions().Create(selected, component.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
		if err != nil {
			t.Fatal(err)
		}
		files, err := client.Resources().UpsertFile(selected, sdk.ResourceFile{ReportID: component.ID, VersionNo: version.VersionNo, Namespace: "owner.docs", ResourcePath: "shared.sql", Content: name, ExpectedSourceRevision: version.SourceRevision})
		if err != nil {
			t.Fatalf("%s shared resource name: %v", name, err)
		}
		snapshot, err := client.Resources().Get(selected, component.ID, version.VersionNo)
		if err != nil || len(snapshot.Files) != 1 || snapshot.Files[0].Content != name {
			t.Fatalf("%s resource crossed: %+v %v", name, snapshot, err)
		}
		items = append(items, authored{selected, component, version, files.Files[0].ResourceID, files.Version.SourceRevision})
	}
	var claims int
	if err = db.QueryRow(`SELECT COUNT(*) FROM resource_namespace_claims WHERE namespace='owner.docs'`).Scan(&claims); err != nil || claims != 2 {
		t.Fatalf("independent claims=%d err=%v", claims, err)
	}
	if _, err = client.Resources().Get(items[0].ctx, items[1].component.ID, items[1].version.VersionNo); err == nil {
		t.Fatal("foreign workspace resource readable")
	}
	if _, err = client.Resources().DeleteFileWithRevision(items[0].ctx, sdk.ResourceDeleteInput{ReportID: items[0].component.ID, VersionNo: items[0].version.VersionNo, ResourceID: items[0].fileID, ExpectedSourceRevision: items[0].revision}); err != nil {
		t.Fatal(err)
	}
	remaining, err := client.Resources().Get(items[1].ctx, items[1].component.ID, items[1].version.VersionNo)
	if err != nil || len(remaining.Files) != 1 || remaining.Files[0].Content != "beta" {
		t.Fatalf("releasing Alpha affected Beta: %+v %v", remaining, err)
	}

}
