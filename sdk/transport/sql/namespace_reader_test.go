package sqltransport

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	_ "modernc.org/sqlite"
)

func TestNamespaceStoreReadScopeAndPaging(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db, Authorizer: allowAuthorizer{}}
	client, err := sdk.NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	owner := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner"})
	viewer := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "viewer"})
	for _, row := range []struct{ name, title, description, status string }{
		{"alpha", "Finance Alpha", "", "active"},
		{"beta", "Other", "Finance beta", "active"},
		{"gamma", "Finance Gamma", "", "archived"},
	} {
		created, createErr := client.Namespaces().Create(owner, sdk.CreateNamespaceInput{Name: row.name, Title: row.title, Description: row.description})
		if createErr != nil {
			t.Fatal(createErr)
		}
		if row.status == "archived" {
			if _, err := db.Exec(`UPDATE namespaces SET status='archived' WHERE owner_id='owner' AND name=?`, row.name); err != nil {
				t.Fatal(err)
			}
		}
		if created.OwnerID != "owner" || created.ETag != 1 || created.CreatedAt.IsZero() {
			t.Fatalf("created=%+v", created)
		}
	}
	if _, err := client.Namespaces().Create(viewer, sdk.CreateNamespaceInput{Name: "private", Title: "Finance Private"}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for index, name := range []string{"alpha", "beta", "gamma"} {
		if _, err := db.Exec(`UPDATE namespaces SET updated_at=? WHERE owner_id='owner' AND name=?`, now.Add(time.Duration(index)*time.Hour), name); err != nil {
			t.Fatal(err)
		}
	}
	page, err := client.Namespaces().List(owner, sdk.ListNamespacesInput{Query: "FINANCE", Status: "active", Limit: 1, Offset: 0})
	if err != nil || page.Limit != 1 || page.Offset != 0 || len(page.Items) != 1 || page.Items[0].Name != "beta" || page.Items[0].Description != "Finance beta" {
		t.Fatalf("first page=%+v err=%v", page, err)
	}
	page, err = client.Namespaces().List(owner, sdk.ListNamespacesInput{Query: "finance", Status: "active", Limit: 1, Offset: 1})
	if err != nil || page.Offset != 1 || len(page.Items) != 1 || page.Items[0].Name != "alpha" {
		t.Fatalf("second page=%+v err=%v", page, err)
	}
	page, err = client.Namespaces().List(viewer, sdk.ListNamespacesInput{})
	if err != nil || len(page.Items) != 1 || page.Items[0].Name != "private" || page.Limit != 50 {
		t.Fatalf("viewer page=%+v err=%v", page, err)
	}
	if _, err := client.Namespaces().Get(viewer, "alpha"); err == nil {
		t.Fatal("viewer read owner namespace without ACL")
	} else {
		var sdkErr *sdk.Error
		if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorNotFound {
			t.Fatalf("missing namespace error=%v", err)
		}
	}
	if _, err := client.Namespaces().Get(owner, ""); err == nil {
		t.Fatal("empty namespace name returned a row")
	} else {
		var sdkErr *sdk.Error
		if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorNotFound {
			t.Fatalf("empty name error=%v", err)
		}
	}
	connector, err := client.Connectors().Create(owner, sdk.CreateConnectorInput{Name: "main", Driver: "sqlite"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE connectors SET status='active' WHERE name=?`, connector.Name); err != nil {
		t.Fatal(err)
	}
	report, err := client.Reports().Create(owner, sdk.CreateReportInput{Slug: "alpha", Title: "Alpha", Namespace: "alpha", DefaultConnectorName: connector.Name})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO report_acl(report_id,subject_type,subject_id,can_view,can_run,can_edit,can_publish,can_use_dql,etag) VALUES(?,'user','viewer',TRUE,FALSE,FALSE,FALSE,FALSE,1)`, report.ID); err != nil {
		t.Fatal(err)
	}
	visible, err := client.Namespaces().Get(viewer, "alpha")
	if err != nil || visible.OwnerID != "owner" || visible.Name != "alpha" {
		t.Fatalf("ACL namespace=%+v err=%v", visible, err)
	}
	page, err = client.Namespaces().List(viewer, sdk.ListNamespacesInput{Query: "finance"})
	if err != nil || len(page.Items) != 2 || page.Items[0].Name != "alpha" || page.Items[1].Name != "private" {
		t.Fatalf("ACL page=%+v err=%v", page, err)
	}
	if _, err := db.Exec(`UPDATE reports SET deleted_at=? WHERE id=?`, now, report.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Namespaces().Get(viewer, "alpha"); err == nil {
		t.Fatal("deleted report still granted namespace access")
	}
	page, err = client.Namespaces().List(ctx, sdk.ListNamespacesInput{Status: "archived", Limit: 0, Offset: -1})
	if err != nil || len(page.Items) != 1 || page.Items[0].Name != "gamma" || page.Offset != 0 || page.Limit != 50 {
		t.Fatalf("unscoped archived page=%+v err=%v", page, err)
	}
}
