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

func TestNamespaceNativeWritesRejectStaleTokens(t *testing.T) {
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
	created, err := client.Namespaces().Create(owner, sdk.CreateNamespaceInput{Name: "finance", Title: "Finance"})
	if err != nil || created.ETag != 1 {
		t.Fatalf("create=%+v err=%v", created, err)
	}
	other := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "other"})
	otherNamespace, err := client.Namespaces().Create(other, sdk.CreateNamespaceInput{Name: "finance", Title: "Other finance"})
	if err != nil {
		t.Fatal(err)
	}
	archived := "archived"
	if _, err := client.Namespaces().Update(other, "finance", sdk.UpdateNamespaceInput{Status: &archived, ETag: otherNamespace.ETag}); err != nil {
		t.Fatal(err)
	}
	if err := transport.requireActiveNamespace(ctx, "owner", "finance"); err != nil {
		t.Fatalf("owner's active namespace hidden by other owner: %v", err)
	}
	if err := transport.requireActiveNamespace(ctx, "other", "finance"); !namespaceCode(err, sdk.ErrorInvalidArgument) {
		t.Fatalf("other owner's archived namespace accepted: %v", err)
	}
	if _, err := client.Namespaces().Create(owner, sdk.CreateNamespaceInput{Name: "finance", Title: "Duplicate"}); !namespaceCode(err, sdk.ErrorConflict) {
		t.Fatalf("duplicate create error=%v", err)
	}
	title := "Finance updated"
	updated, err := client.Namespaces().Update(owner, created.Name, sdk.UpdateNamespaceInput{Title: &title, ETag: created.ETag})
	if err != nil || updated.ETag != 2 || updated.Title != title {
		t.Fatalf("update=%+v err=%v", updated, err)
	}
	stale := "Stale title"
	if _, err := client.Namespaces().Update(owner, created.Name, sdk.UpdateNamespaceInput{Title: &stale, ETag: created.ETag}); !namespaceCode(err, sdk.ErrorConflict) {
		t.Fatalf("stale update error=%v", err)
	}
	if err := client.Namespaces().Delete(owner, created.Name, created.ETag); !namespaceCode(err, sdk.ErrorConflict) {
		t.Fatalf("stale delete error=%v", err)
	}
	current, err := client.Namespaces().Get(owner, created.Name)
	if err != nil || current.Title != title || current.ETag != updated.ETag {
		t.Fatalf("stale write changed row=%+v err=%v", current, err)
	}
	if err := client.Namespaces().Delete(owner, created.Name, updated.ETag); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Namespaces().Get(owner, created.Name); !namespaceCode(err, sdk.ErrorNotFound) {
		t.Fatalf("soft-deleted namespace visible: %v", err)
	}
	var status string
	var etag int64
	var deleted sql.NullTime
	if err := db.QueryRowContext(ctx, `SELECT status,etag,deleted_at FROM namespaces WHERE owner_id=? AND name=?`, "owner", created.Name).Scan(&status, &etag, &deleted); err != nil {
		t.Fatal(err)
	}
	if status != "archived" || etag != 3 || !deleted.Valid {
		t.Fatalf("soft delete status=%q etag=%d deleted=%v", status, etag, deleted.Valid)
	}
}

func namespaceCode(err error, code sdk.ErrorCode) bool {
	var actual *sdk.Error
	return errors.As(err, &actual) && actual.Code == code
}

func TestConfiguredNamespaceSettingsDenyOwnerBeforeWriting(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db, Authorizer: allowAuthorizer{}}
	client, err := sdk.NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	owner := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner"})
	created, err := client.Namespaces().Create(owner, sdk.CreateNamespaceInput{Name: "finance", Title: "Finance"})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("STUDIO_NAMESPACE_CREATE_ROLES", "APP_ADMIN,DATA_ADMIN")
	title := "Unauthorized"
	if _, err = client.Namespaces().Update(owner, "finance", sdk.UpdateNamespaceInput{Title: &title, ETag: created.ETag}); !namespaceCode(err, sdk.ErrorForbidden) {
		t.Fatalf("owner update: %v", err)
	}
	if err = client.Namespaces().Delete(owner, "finance", created.ETag); !namespaceCode(err, sdk.ErrorForbidden) {
		t.Fatalf("owner delete: %v", err)
	}
	if _, err = client.Namespaces().Create(owner, sdk.CreateNamespaceInput{Name: "another", Title: "Another"}); !namespaceCode(err, sdk.ErrorForbidden) {
		t.Fatalf("owner create: %v", err)
	}
	var storedTitle string
	var etag, count int
	if err = db.QueryRowContext(ctx, "SELECT title,etag FROM namespaces WHERE name='finance'").Scan(&storedTitle, &etag); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM namespaces").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if storedTitle != "Finance" || etag != 1 || count != 1 {
		t.Fatalf("denied operation changed storage: %s/%d/%d", storedTitle, etag, count)
	}
}
