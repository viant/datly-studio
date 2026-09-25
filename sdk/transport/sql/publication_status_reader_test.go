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
	_ "modernc.org/sqlite"
)

func TestPublicationStatusReaderPreservesNullableFields(t *testing.T) {
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
	report, err := client.Reports().Create(owner, sdk.CreateReportInput{Slug: "publication-read", Title: "Publication read", DefaultConnectorName: connector.Name})
	if err != nil {
		t.Fatal(err)
	}
	version, err := client.Versions().Create(owner, report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := db.ExecContext(ctx, `INSERT INTO report_publications
		(report_id,active_version_no,desired_generation,publication_status,spec_hash,published_by,published_at)
		VALUES(?,?,1,'pending',?,'owner',?)`, report.ID, version.VersionNo, version.SpecHash, now); err != nil {
		t.Fatal(err)
	}
	value, err := transport.readPublicationStatus(owner, report.ID)
	if err != nil || value.ReportID != report.ID || value.ActiveVersionNo != 1 ||
		value.DesiredVersionNo != nil || value.ActiveGeneration != nil || value.RuntimeRevision != "" ||
		value.Status != "pending" || value.PublishedAt == nil || value.SpecHash != version.SpecHash {
		t.Fatalf("publication=%+v err=%v", value, err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE report_publications SET publication_status='unpublishing' WHERE report_id=?`, report.ID); err != nil {
		t.Fatal(err)
	}
	snapshot, found, err := transport.publicationSnapshot(owner, tx, report.ID)
	if err != nil || !found || snapshot.status != "unpublishing" || snapshot.publishedBy != "owner" || snapshot.activeVersion != 1 {
		t.Fatalf("transaction publication snapshot=%+v found=%v err=%v", snapshot, found, err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	value, err = transport.readPublicationStatus(owner, report.ID)
	if err != nil || value.Status != "pending" {
		t.Fatalf("rolled-back publication=%+v err=%v", value, err)
	}
	missing := &sdk.Publication{}
	err = transport.getPublication(owner, "missing", missing)
	var sdkErr *sdk.Error
	if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorNotFound {
		t.Fatalf("missing publication error=%v", err)
	}
}
