package sqltransport

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	_ "modernc.org/sqlite"
)

func TestReaderBuilderRollsBackVersionWhenReportWriteFails(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
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
	for _, name := range []string{"main", "lookup"} {
		connector, createErr := client.Connectors().Create(owner, sdk.CreateConnectorInput{Name: name, Driver: "sqlite"})
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, err := db.Exec(`UPDATE connectors SET status='active' WHERE name=?`, connector.Name); err != nil {
			t.Fatal(err)
		}
	}
	report, err := client.Reports().Create(owner, sdk.CreateReportInput{Slug: "builder-rollback", Title: "Builder rollback", DefaultConnectorName: "main"})
	if err != nil {
		t.Fatal(err)
	}
	version, err := client.Versions().Create(owner, report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TRIGGER reject_report_update BEFORE UPDATE ON reports BEGIN SELECT RAISE(ABORT,'report write failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := transport.persistReaderBuilderDQL(owner, report.ID, version, "SELECT 2", "lookup", false); err == nil {
		t.Fatal("report write failure did not abort Reader Builder transaction")
	}
	current, err := transport.getVersionValue(owner, report.ID, version.VersionNo)
	if err != nil || current.SourceRevision != version.SourceRevision || current.AuthoredDQL != "SELECT 1" {
		t.Fatalf("version changed despite rollback: %+v err=%v", current, err)
	}
	storedReport, err := transport.getReportValue(owner, report.ID)
	if err != nil || storedReport.DefaultConnectorName != "main" || storedReport.ETag != report.ETag {
		t.Fatalf("report changed despite rollback: %+v err=%v", storedReport, err)
	}
	if _, err := db.Exec(`DROP TRIGGER reject_report_update`); err != nil {
		t.Fatal(err)
	}
	updated, err := transport.persistReaderBuilderDQL(owner, report.ID, version, "SELECT 2", "lookup", false)
	if err != nil || updated.SourceRevision != version.SourceRevision+1 || updated.AuthoredDQL != "SELECT 2" {
		t.Fatalf("successful Reader Builder version=%+v err=%v", updated, err)
	}
	storedReport, err = transport.getReportValue(owner, report.ID)
	if err != nil || storedReport.DefaultConnectorName != "lookup" || storedReport.ETag != report.ETag+1 {
		t.Fatalf("successful Reader Builder report=%+v err=%v", storedReport, err)
	}
}
