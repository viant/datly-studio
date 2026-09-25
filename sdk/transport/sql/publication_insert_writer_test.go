package sqltransport

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	stored "github.com/viant/datly-studio/studio/report_publications/store_insert"
	_ "modernc.org/sqlite"
)

func TestPublicationInsertWriterJoinsGenerationTransaction(t *testing.T) {
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
	connector, err := client.Connectors().Create(owner, sdk.CreateConnectorInput{Name: "main", Driver: "sqlite"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE connectors SET status='active' WHERE name=?`, connector.Name); err != nil {
		t.Fatal(err)
	}
	report, err := client.Reports().Create(owner, sdk.CreateReportInput{Slug: "stage-insert", Title: "Stage insert", DefaultConnectorName: connector.Name})
	if err != nil {
		t.Fatal(err)
	}
	version, err := client.Versions().Create(owner, report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
	if err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	revision := "report:1:1"
	if err := transport.insertBuildingGeneration(owner, tx, 1, revision, "owner", now); err != nil {
		t.Fatal(err)
	}
	versionNo := version.VersionNo
	err = transport.writePublicationInsert(owner, tx, &stored.StoredPublication{
		ReportId: report.ID, ActiveVersionNo: versionNo, DesiredVersionNo: &versionNo,
		DesiredGeneration: 1, PublicationStatus: "pending", RuntimeRevision: &revision,
		SpecHash: version.SpecHash, PublishedBy: "owner", PublishedAt: now,
		Has: &stored.StoredPublicationHas{ReportId: true, ActiveVersionNo: true,
			DesiredVersionNo: true, DesiredGeneration: true, ActiveGeneration: true,
			PublicationStatus: true, RuntimeRevision: true, SpecHash: true,
			PublishedBy: true, PublishedAt: true, ActivatedAt: true, FailureJson: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, found, err := transport.publicationSnapshot(owner, tx, report.ID)
	if err != nil || !found || snapshot.status != "pending" || snapshot.desiredGeneration != 1 {
		t.Fatalf("staged publication=%+v found=%v err=%v", snapshot, found, err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if _, err := transport.readPublicationStatus(owner, report.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("rolled-back publication error=%v", err)
	}
	next, err := transport.nextGenerationNo(owner, nil)
	if err != nil || next != 1 {
		t.Fatalf("rolled-back generation=%d err=%v", next, err)
	}
}
