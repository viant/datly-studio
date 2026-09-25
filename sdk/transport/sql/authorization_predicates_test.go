package sqltransport

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

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

func TestAuthorizationPredicateDatlyStoreReadPreservesFiltersAndPage(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	catalog, err := predicatecatalog.New(predicatecatalog.Package{Alias: "fixture", Path: "example.com/iam", Types: []reflect.Type{reflect.TypeFor[dynamicPredicateFixture]()}})
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 9, 24, 9, 0, 0, 0, time.UTC)
	for _, row := range []struct {
		name, title, description, path, typeName, status, scope string
		updated, deleted                                        any
	}{
		{"iam.alpha.read", "Alpha", "needle in description", "example.com/iam", "dynamicPredicateFixture", "active", `{"alias":"a","columns":["tenant_id"]}`, base.Add(time.Hour), nil},
		{"iam.beta.read", "NEEDLE title", "", "example.com/beta", "Beta", "active", "", base.Add(time.Hour), nil},
		{"iam.gamma.read", "Gamma", "", "example.com/needle", "Gamma", "disabled", "", base, nil},
		{"iam.deleted.read", "Deleted", "needle", "example.com/deleted", "Deleted", "active", "", base.Add(2 * time.Hour), base.Add(3 * time.Hour)},
	} {
		if _, err := db.ExecContext(ctx, `INSERT INTO authorization_predicates(name,title,description,package_path,type_name,sql_scope_json,owner_id,status,etag,created_at,updated_at,deleted_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
			row.name, row.title, nullable(row.description), row.path, row.typeName, nullable(row.scope), "owner", row.status, 7, base, row.updated, row.deleted); err != nil {
			t.Fatal(err)
		}
	}
	client, err := sdk.NewClient(&Transport{DB: db, Authorizer: allowAuthorizer{}, Predicates: catalog})
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.AuthorizationPredicates().Get(ctx, "iam.alpha.read")
	if err != nil {
		t.Fatal(err)
	}
	if got.Description != "needle in description" || got.Alias != "a" || len(got.Columns) != 1 || got.Columns[0] != "tenant_id" || !got.Linked || got.ETag != 7 || !got.CreatedAt.Equal(base) {
		t.Fatalf("get=%+v", got)
	}
	for _, name := range []string{"", "iam.missing.read", "iam.deleted.read"} {
		_, err := client.AuthorizationPredicates().Get(ctx, name)
		var sdkErr *sdk.Error
		if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorNotFound {
			t.Fatalf("get %q error=%v", name, err)
		}
	}
	for _, tc := range []struct {
		input sdk.ListAuthorizationPredicatesInput
		want  []string
	}{
		{sdk.ListAuthorizationPredicatesInput{}, []string{"iam.alpha.read", "iam.beta.read", "iam.gamma.read"}},
		{sdk.ListAuthorizationPredicatesInput{Query: "  NEEDLE  "}, []string{"iam.alpha.read", "iam.beta.read", "iam.gamma.read"}},
		{sdk.ListAuthorizationPredicatesInput{Query: "gamma"}, []string{"iam.gamma.read"}},
		{sdk.ListAuthorizationPredicatesInput{Query: "beta"}, []string{"iam.beta.read"}},
		{sdk.ListAuthorizationPredicatesInput{Query: "needle", Status: "active"}, []string{"iam.alpha.read", "iam.beta.read"}},
		{sdk.ListAuthorizationPredicatesInput{Status: "disabled"}, []string{"iam.gamma.read"}},
		{sdk.ListAuthorizationPredicatesInput{Status: "ACTIVE"}, nil},
		{sdk.ListAuthorizationPredicatesInput{Limit: 1, Offset: 1}, []string{"iam.beta.read"}},
		{sdk.ListAuthorizationPredicatesInput{Limit: 900, Offset: -5}, []string{"iam.alpha.read", "iam.beta.read", "iam.gamma.read"}},
	} {
		page, err := client.AuthorizationPredicates().List(ctx, tc.input)
		if err != nil {
			t.Fatalf("list %+v: %v", tc.input, err)
		}
		if len(page.Items) != len(tc.want) {
			t.Fatalf("list %+v: got=%+v want=%v", tc.input, page, tc.want)
		}
		for i, item := range page.Items {
			if item.Name != tc.want[i] {
				t.Fatalf("list %+v item %d=%s want=%s", tc.input, i, item.Name, tc.want[i])
			}
		}
		if tc.input.Limit == 900 && (page.Limit != 500 || page.Offset != 0) {
			t.Fatalf("clamped page=%+v", page)
		}
		if tc.input.Limit == 0 && page.Limit != 50 {
			t.Fatalf("default page limit=%d", page.Limit)
		}
	}
}

func TestAuthorizationPredicateStoreAuthorizesBeforeRead(t *testing.T) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	client, err := sdk.NewClient(&Transport{DB: db, Authorizer: AuthorizerFunc(func(context.Context, AuthorizationRequest) error {
		return errors.New("denied")
	})})
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range []func() error{
		func() error {
			_, err := client.AuthorizationPredicates().Get(context.Background(), "iam.alpha.read")
			return err
		},
		func() error {
			_, err := client.AuthorizationPredicates().List(context.Background(), sdk.ListAuthorizationPredicatesInput{})
			return err
		},
	} {
		var sdkErr *sdk.Error
		if err := operation(); !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorForbidden {
			t.Fatalf("read should fail authorization first: %v", err)
		}
	}
}

func TestAuthorizationPredicateReaderReusesRuntimeAndSeesFreshRows(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	catalog, err := predicatecatalog.New(predicatecatalog.Package{Alias: "fixture", Path: "example.com/iam", Types: []reflect.Type{reflect.TypeFor[dynamicPredicateFixture]()}})
	if err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db, Authorizer: allowAuthorizer{}, Predicates: catalog}
	defer transport.Close(ctx)
	client, err := sdk.NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	first, err := client.AuthorizationPredicates().List(ctx, sdk.ListAuthorizationPredicatesInput{})
	if err != nil || len(first.Items) != 0 || transport.predicateReader == nil {
		t.Fatalf("initial page=%+v reader=%v err=%v", first, transport.predicateReader, err)
	}
	initialReader := transport.predicateReader
	now := time.Now().UTC()
	_, err = db.ExecContext(ctx, `INSERT INTO authorization_predicates(name,title,package_path,type_name,owner_id,status,etag,created_at,updated_at)
		VALUES('iam.new.read','New','example.com/iam','dynamicPredicateFixture','owner','active',1,?,?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := client.AuthorizationPredicates().List(ctx, sdk.ListAuthorizationPredicatesInput{})
	if err != nil || len(second.Items) != 1 || second.Items[0].Name != "iam.new.read" || transport.predicateReader != initialReader {
		t.Fatalf("fresh page=%+v reused=%v err=%v", second, transport.predicateReader == initialReader, err)
	}
	if err := transport.Close(ctx); err != nil || transport.predicateReader != nil {
		t.Fatalf("reader close=%v err=%v", transport.predicateReader, err)
	}
}

func TestAuthorizationPredicateDatlyStoreWriteLifecycle(t *testing.T) {
	ctx := sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "owner"})
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	catalog, err := predicatecatalog.New(predicatecatalog.Package{Alias: "fixture", Path: "example.com/iam", Types: []reflect.Type{reflect.TypeFor[dynamicPredicateFixture]()}})
	if err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db, Authorizer: allowAuthorizer{}, Predicates: catalog}
	defer transport.Close(ctx)
	client, err := sdk.NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	created, err := client.AuthorizationPredicates().Create(ctx, sdk.CreateAuthorizationPredicateInput{
		Name: "iam.vendor.read", Title: "Vendor access", PackagePath: "example.com/iam", TypeName: "dynamicPredicateFixture",
		Alias: "v", Columns: []string{" vendor_id "},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ETag != 1 || created.OwnerID != "owner" || created.Alias != "v" || len(created.Columns) != 1 || created.Columns[0] != "vendor_id" {
		t.Fatalf("created=%+v", created)
	}
	if _, err := client.AuthorizationPredicates().Create(ctx, sdk.CreateAuthorizationPredicateInput{
		Name: "not-a-canonical-name", Title: "Invalid", PackagePath: created.PackagePath, TypeName: created.TypeName,
	}); !predicateErrorCode(err, sdk.ErrorInvalidArgument) {
		t.Fatalf("invalid create=%v", err)
	}
	if _, err := client.AuthorizationPredicates().Create(ctx, sdk.CreateAuthorizationPredicateInput{
		Name: created.Name, Title: "duplicate", PackagePath: created.PackagePath, TypeName: created.TypeName,
	}); !predicateErrorCode(err, sdk.ErrorConflict) {
		t.Fatalf("duplicate create=%v", err)
	}
	newTitle, empty := " Updated ", ""
	updated, err := client.AuthorizationPredicates().Update(ctx, created.Name, sdk.UpdateAuthorizationPredicateInput{
		ETag: created.ETag, Title: &newTitle, Alias: &empty,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ETag != 2 || updated.Title != "Updated" || updated.Alias != "" || updated.OwnerID != "owner" {
		t.Fatalf("updated=%+v", updated)
	}
	if _, err := client.AuthorizationPredicates().Update(ctx, created.Name, sdk.UpdateAuthorizationPredicateInput{ETag: updated.ETag, Title: &empty}); !predicateErrorCode(err, sdk.ErrorInvalidArgument) {
		t.Fatalf("invalid update=%v", err)
	}
	afterInvalid, err := client.AuthorizationPredicates().Get(ctx, created.Name)
	if err != nil || afterInvalid.ETag != updated.ETag || afterInvalid.Title != updated.Title {
		t.Fatalf("invalid update changed row: %+v, %v", afterInvalid, err)
	}
	if _, err := client.AuthorizationPredicates().Update(ctx, created.Name, sdk.UpdateAuthorizationPredicateInput{ETag: created.ETag}); !predicateErrorCode(err, sdk.ErrorConflict) {
		t.Fatalf("stale update=%v", err)
	}
	if err := client.AuthorizationPredicates().Delete(ctx, created.Name, created.ETag); !predicateErrorCode(err, sdk.ErrorConflict) {
		t.Fatalf("stale delete=%v", err)
	}
	if err := client.AuthorizationPredicates().Delete(ctx, created.Name, updated.ETag); err != nil {
		t.Fatal(err)
	}
	var status string
	var deletedAt sql.NullTime
	var etag int64
	if err := db.QueryRowContext(ctx, `SELECT status,deleted_at,etag FROM authorization_predicates WHERE name=?`, created.Name).Scan(&status, &deletedAt, &etag); err != nil {
		t.Fatal(err)
	}
	if status != "disabled" || !deletedAt.Valid || etag != 3 {
		t.Fatalf("soft delete: status=%q deleted=%v etag=%d", status, deletedAt, etag)
	}
	if _, err := client.AuthorizationPredicates().Get(ctx, created.Name); !predicateErrorCode(err, sdk.ErrorNotFound) {
		t.Fatalf("get deleted=%v", err)
	}
	if err := client.AuthorizationPredicates().Delete(ctx, created.Name, etag); !predicateErrorCode(err, sdk.ErrorConflict) {
		t.Fatalf("delete twice=%v", err)
	}
}

func predicateErrorCode(err error, code sdk.ErrorCode) bool {
	var sdkErr *sdk.Error
	return errors.As(err, &sdkErr) && sdkErr.Code == code
}

func TestAuthorizationPredicateStoreAuthorizesBeforeMutation(t *testing.T) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	client, err := sdk.NewClient(&Transport{DB: db, Authorizer: AuthorizerFunc(func(context.Context, AuthorizationRequest) error {
		return errors.New("denied")
	})})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, operation := range []func() error{
		func() error {
			_, err := client.AuthorizationPredicates().Create(ctx, sdk.CreateAuthorizationPredicateInput{Name: "iam.vendor.read"})
			return err
		},
		func() error {
			_, err := client.AuthorizationPredicates().Update(ctx, "iam.vendor.read", sdk.UpdateAuthorizationPredicateInput{ETag: 1})
			return err
		},
		func() error { return client.AuthorizationPredicates().Delete(ctx, "iam.vendor.read", 1) },
	} {
		if err := operation(); !predicateErrorCode(err, sdk.ErrorForbidden) {
			t.Fatalf("mutation should fail authorization first: %v", err)
		}
	}
}
