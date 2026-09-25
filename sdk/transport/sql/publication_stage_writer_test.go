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
	publicationdelete "github.com/viant/datly-studio/studio/report_publications/store_delete"
	publicationrecover "github.com/viant/datly-studio/studio/report_publications/store_recover"
	stored "github.com/viant/datly-studio/studio/report_publications/store_stage"
	xhandler "github.com/viant/xdatly/handler"
	_ "modernc.org/sqlite"
)

func TestPublicationRestageWriterMatchesGenerationAndRollsBack(t *testing.T) {
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
	report, err := client.Reports().Create(owner, sdk.CreateReportInput{Slug: "restage", Title: "Restage", DefaultConnectorName: connector.Name})
	if err != nil {
		t.Fatal(err)
	}
	version, err := client.Versions().Create(owner, report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := transport.insertBuildingGeneration(owner, nil, 1, "report:1:1", "owner", now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE runtime_generations SET status='active',activated_at=? WHERE generation_no=1`, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO report_publications
		(report_id,active_version_no,desired_version_no,desired_generation,active_generation,publication_status,
		 runtime_revision,spec_hash,published_by,published_at,activated_at)
		VALUES(?,?,?,1,1,'active','report:1:1',?,'owner',?,?)`, report.ID, version.VersionNo, version.VersionNo, version.SpecHash, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE report_publications SET failure_json='[{"code":"previous"}]' WHERE report_id=?`, report.ID); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err := transport.insertBuildingGeneration(owner, tx, 2, "report:1:2", "owner", now); err != nil {
		t.Fatal(err)
	}
	expected := int64(1)
	versionNo := version.VersionNo
	revision := "report:1:2"
	row := &stored.StoredPublication{ReportId: report.ID, DesiredVersionNo: &versionNo,
		DesiredGeneration: &expected, PublicationStatus: "pending", RuntimeRevision: &revision,
		SpecHash: version.SpecHash, PublishedBy: "owner", PublishedAt: &now,
		Has: &stored.StoredPublicationHas{ReportId: true, DesiredVersionNo: true,
			DesiredGeneration: true, PublicationStatus: true, RuntimeRevision: true,
			SpecHash: true, PublishedBy: true, PublishedAt: true, FailureJson: true}}
	if err := transport.writePublicationStage(owner, tx, "publish", 2, row); err != nil {
		t.Fatal(err)
	}
	snapshot, found, err := transport.publicationSnapshot(owner, tx, report.ID)
	if err != nil || !found || snapshot.desiredGeneration != 2 || snapshot.status != "pending" ||
		!snapshot.activeGeneration.Valid || snapshot.activeGeneration.Int64 != 1 || snapshot.activeVersion != 1 {
		t.Fatalf("staged snapshot=%+v found=%v err=%v", snapshot, found, err)
	}
	var failure sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT failure_json FROM report_publications WHERE report_id=?`, report.ID).Scan(&failure); err != nil || failure.Valid {
		t.Fatalf("restaged failure_json=%v err=%v, want SQL NULL", failure, err)
	}
	staleExpected := int64(1)
	stale := *row
	stale.DesiredGeneration = &staleExpected
	var conflict *xhandler.Conflict
	if err := transport.writePublicationStage(owner, tx, "publish", 3, &stale); !errors.As(err, &conflict) {
		t.Fatalf("stale restage error=%v", err)
	}
	restoreGeneration := int64(1)
	restoreVersion := version.VersionNo
	restoreRevision := "report:1:1"
	diagnostics := `[{"code":"reload_failed"}]`
	restoreRow := &publicationrecover.StoredPublication{ReportId: report.ID,
		ActiveVersionNo: version.VersionNo, DesiredVersionNo: &restoreVersion,
		DesiredGeneration: &restoreGeneration, ActiveGeneration: &restoreGeneration,
		PublicationStatus: "pending", RuntimeRevision: &restoreRevision,
		SpecHash: version.SpecHash, PublishedBy: "owner", PublishedAt: &now,
		ActivatedAt: &now, FailureJson: &diagnostics,
		Has: &publicationrecover.StoredPublicationHas{ReportId: true, ActiveVersionNo: true,
			DesiredVersionNo: true, DesiredGeneration: true, ActiveGeneration: true,
			PublicationStatus: true, RuntimeRevision: true, SpecHash: true,
			PublishedBy: true, PublishedAt: true, ActivatedAt: true, FailureJson: true}}
	if err := transport.writePublicationCompensation(owner, tx, "compensate_restore", 3, "active", restoreRow); !errors.As(err, &conflict) {
		t.Fatalf("stale compensation generation error=%v", err)
	}
	if err := transport.writePublicationCompensation(owner, tx, "compensate_restore", 2, "active", restoreRow); err != nil {
		t.Fatalf("matched compensation: %v", err)
	}
	snapshot, found, err = transport.publicationSnapshot(owner, tx, report.ID)
	if err != nil || !found || snapshot.status != "active" || snapshot.desiredGeneration != 1 ||
		!snapshot.activeGeneration.Valid || snapshot.activeGeneration.Int64 != 1 || snapshot.specHash != version.SpecHash {
		t.Fatalf("compensated snapshot=%+v found=%v err=%v", snapshot, found, err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	snapshot, found, err = transport.publicationSnapshot(owner, nil, report.ID)
	if err != nil || !found || snapshot.desiredGeneration != 1 || snapshot.status != "active" {
		t.Fatalf("rolled-back snapshot=%+v found=%v err=%v", snapshot, found, err)
	}
	next, err := transport.nextGenerationNo(owner, nil)
	if err != nil || next != 2 {
		t.Fatalf("rolled-back generation head=%d err=%v", next, err)
	}
	unpublishTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer unpublishTx.Rollback()
	if err := transport.insertBuildingGeneration(owner, unpublishTx, 2, "unpublish:report:2", "owner", now); err != nil {
		t.Fatal(err)
	}
	expected = 1
	if err := transport.writePublicationStage(owner, unpublishTx, "unpublish", 2, &stored.StoredPublication{
		ReportId: report.ID, DesiredGeneration: &expected, PublicationStatus: "unpublishing",
		Has: &stored.StoredPublicationHas{ReportId: true, DesiredVersionNo: true,
			DesiredGeneration: true, PublicationStatus: true, FailureJson: true},
	}); err != nil {
		t.Fatal(err)
	}
	var desiredVersion sql.NullInt64
	var desiredGeneration int64
	var stageStatus string
	if err := unpublishTx.QueryRowContext(ctx, `SELECT desired_version_no,desired_generation,publication_status,failure_json FROM report_publications WHERE report_id=?`, report.ID).
		Scan(&desiredVersion, &desiredGeneration, &stageStatus, &failure); err != nil || desiredVersion.Valid || desiredGeneration != 2 || stageStatus != "unpublishing" || failure.Valid {
		t.Fatalf("unpublish stage version=%v generation=%d status=%q failure=%v err=%v", desiredVersion, desiredGeneration, stageStatus, failure, err)
	}
	staleGeneration := int64(1)
	deleteRow := func(generation *int64) *publicationdelete.StoredPublication {
		return &publicationdelete.StoredPublication{ReportId: report.ID, DesiredGeneration: generation, ShouldDelete: true,
			Has: &publicationdelete.StoredPublicationHas{ReportId: true, DesiredGeneration: true, ShouldDelete: true}}
	}
	var deleteConflict *xhandler.Conflict
	if err := transport.writePublicationDelete(owner, unpublishTx, deleteRow(&staleGeneration)); !errors.As(err, &deleteConflict) {
		t.Fatalf("stale unpublish delete error=%v", err)
	}
	if err := transport.writePublicationDelete(owner, unpublishTx, deleteRow(&desiredGeneration)); err != nil {
		t.Fatalf("matched unpublish delete: %v", err)
	}
	var remaining int
	if err := unpublishTx.QueryRowContext(ctx, `SELECT COUNT(1) FROM report_publications WHERE report_id=?`, report.ID).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatalf("deleted publication remaining=%d err=%v", remaining, err)
	}
	if err := unpublishTx.Rollback(); err != nil {
		t.Fatal(err)
	}
	snapshot, found, err = transport.publicationSnapshot(owner, nil, report.ID)
	if err != nil || !found || snapshot.desiredGeneration != 1 || snapshot.status != "active" {
		t.Fatalf("rolled-back unpublish=%+v found=%v err=%v", snapshot, found, err)
	}
}
