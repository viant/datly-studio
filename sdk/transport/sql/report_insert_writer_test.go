package sqltransport

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	stored "github.com/viant/datly-studio/studio/reports/store_insert"
	xhandler "github.com/viant/xdatly/handler"
	"path/filepath"
)

func TestReportInsertReservesDeletedIdentity(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(context.Background(), db, "studio"); err != nil {
		t.Fatal(err)
	}
	testReportInsertReceiptAdmission(context.Background(), t, db)
}
func TestReportInsertReservesDeletedIdentityMySQL(t *testing.T) {
	dsn := os.Getenv("STUDIO_REMOVAL_MYSQL_DSN")
	if dsn == "" {
		t.Skip("isolated MySQL fixture is required")
	}
	ctx := context.Background()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.Initialize(ctx, db, "mysql"); err != nil {
		t.Fatal(err)
	}
	testReportInsertReceiptAdmission(ctx, t, db)
	// A concurrent deletion commits its receipt before a waiting creator can
	// admit the same ID. The admission read must observe that committed receipt.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM components WHERE id='fresh'`); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO component_deletions VALUES('fresh','namespace','owner',CURRENT_TIMESTAMP,'owner')`); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	raceCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	go func() {
		client, err := sdk.NewClient(&Transport{DB: db, Authorizer: allowAuthorizer{}})
		if err == nil {
			_, err = client.Components().Create(sdk.WithPrincipal(raceCtx, sdk.Principal{Subject: "owner"}), sdk.CreateComponentInput{ID: "fresh", Slug: "racing", Title: "Example", DefaultConnectorName: "admission"})
		}
		done <- err
	}()
	select {
	case err := <-done:
		t.Fatalf("creation did not wait for pending receipt: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	err = <-done
	var sdkErr *sdk.Error
	if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorConflict {
		t.Fatalf("concurrent retired identity create=%v", err)
	}
	var count int
	if err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM components WHERE id='fresh'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("concurrent ID reuse count=%d err=%v", count, err)
	}

}
func testReportInsertReceiptAdmission(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()
	for _, statement := range []string{
		`INSERT INTO connectors(name,driver,owner_id,status,created_at,updated_at) VALUES('admission','sqlite','owner','active',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO namespaces(owner_id,name,title,status,created_at,updated_at) VALUES('owner','general','General','active',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO component_deletions(component_id,namespace_id,owner_id,deleted_at,deleted_by) VALUES('retired','namespace','former',CURRENT_TIMESTAMP,'former')`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	transport := &Transport{DB: db, Authorizer: allowAuthorizer{}}
	now := time.Now().UTC()
	row := func(id string) *stored.StoredReport {
		return &stored.StoredReport{Id: id, Namespace: "general", Slug: id, Title: "Example", OwnerId: "owner", Status: "draft", DefaultConnectorName: "admission", ComponentScope: "example/components/" + id, ComponentName: "reader", Etag: 1, CreatedAt: now, UpdatedAt: now, Has: &stored.StoredReportHas{Id: true, Namespace: true, Slug: true, Title: true, OwnerId: true, Status: true, DefaultConnectorName: true, ComponentScope: true, ComponentName: true, Etag: true, CreatedAt: true, UpdatedAt: true}}
	}
	if err := transport.writeReportInsert(ctx, row("fresh")); err != nil {
		t.Fatalf("native insert: %v", err)
	}
	err := transport.writeReportInsert(ctx, row("retired"))
	var conflict *xhandler.Conflict
	if !errors.As(err, &conflict) {
		t.Fatalf("native receipt admission=%v, want conflict", err)
	}
	client, err := sdk.NewClient(transport)
	if err != nil {
		t.Fatal(err)
	}
	principal := sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner"})
	_, err = client.Components().Create(principal, sdk.CreateComponentInput{ID: "retired", Slug: "reused", Title: "Example", DefaultConnectorName: "admission"})
	var sdkErr *sdk.Error
	if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorConflict {
		t.Fatalf("SDK receipt admission=%v, want conflict", err)
	}
	var count int
	if err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM components WHERE id='retired'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("reserved identity inserted count=%d err=%v", count, err)
	}
	if err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_deletions WHERE component_id='retired' AND owner_id='former'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("receipt changed count=%d err=%v", count, err)
	}
}
