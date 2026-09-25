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
	activated "github.com/viant/datly-studio/studio/report_publications/store_activate"
	repoint "github.com/viant/datly-studio/studio/report_publications/store_repoint"
	publicationstage "github.com/viant/datly-studio/studio/report_publications/store_stage"
	xhandler "github.com/viant/xdatly/handler"
	_ "modernc.org/sqlite"
)

func TestOtherActivePublicationsRepointAndRollbackTogether(t *testing.T) {
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
	now := time.Now().UTC()
	if err := transport.insertBuildingGeneration(owner, nil, 1, "rev1", "owner", now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE runtime_generations SET status='active',activated_at=? WHERE generation_no=1`, now); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"current", "other"} {
		report, createErr := client.Reports().Create(owner, sdk.CreateReportInput{ID: id, Slug: id, Title: id, DefaultConnectorName: connector.Name})
		if createErr != nil {
			t.Fatal(createErr)
		}
		version, versionErr := client.Versions().Create(owner, report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
		if versionErr != nil {
			t.Fatal(versionErr)
		}
		status, desired := "active", int64(1)
		if id == "current" {
			status, desired = "pending", 2
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO report_publications
			(report_id,active_version_no,desired_version_no,desired_generation,active_generation,publication_status,
			 runtime_revision,spec_hash,published_by,published_at)
			VALUES(?,?,?, ?,1, ?, 'rev1',?,'owner',?)`, id, version.VersionNo, version.VersionNo, desired, status, version.SpecHash, now); err != nil {
			t.Fatal(err)
		}
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err := transport.insertBuildingGeneration(owner, tx, 2, "rev2", "owner", now); err != nil {
		t.Fatal(err)
	}
	expected := int64(2)
	if err := transport.writePublicationActivation(owner, tx, 1, 2, &activated.StoredPublication{ReportId: "current",
		DesiredGeneration: &expected, ActivatedAt: &now,
		Has: &activated.StoredPublicationHas{ReportId: true, DesiredGeneration: true, ActivatedAt: true}}); err != nil {
		t.Fatal(err)
	}
	others, err := transport.readOtherActivePublications(owner, tx, "current")
	if err != nil || len(others) != 1 || others[0].ReportId != "other" {
		t.Fatalf("other active rows=%+v err=%v", others, err)
	}
	old := others[0].ActiveGeneration
	if err := transport.writePublicationRepoint(owner, tx, "current", 2, []*repoint.StoredPublication{{
		ReportId: "other", ActiveGeneration: old,
		Has: &repoint.StoredPublicationHas{ReportId: true, ActiveGeneration: true}}}); err != nil {
		t.Fatal(err)
	}
	stale := int64(1)
	var conflict *xhandler.Conflict
	if err := transport.writePublicationRepoint(owner, tx, "current", 3, []*repoint.StoredPublication{{
		ReportId: "other", ActiveGeneration: &stale,
		Has: &repoint.StoredPublicationHas{ReportId: true, ActiveGeneration: true}}}); !errors.As(err, &conflict) {
		t.Fatalf("stale active generation error=%v", err)
	}
	for _, id := range []string{"current", "other"} {
		snapshot, found, readErr := transport.publicationSnapshot(owner, tx, id)
		if readErr != nil || !found || snapshot.status != "active" || !snapshot.activeGeneration.Valid || snapshot.activeGeneration.Int64 != 2 {
			t.Fatalf("repointed %s=%+v found=%v err=%v", id, snapshot, found, readErr)
		}
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	current, _, err := transport.publicationSnapshot(owner, nil, "current")
	if err != nil || current.status != "pending" || current.activeGeneration.Int64 != 1 {
		t.Fatalf("rolled-back current=%+v err=%v", current, err)
	}
	other, _, err := transport.publicationSnapshot(owner, nil, "other")
	if err != nil || other.status != "active" || other.activeGeneration.Int64 != 1 {
		t.Fatalf("rolled-back other=%+v err=%v", other, err)
	}
	if err := transport.insertBuildingGeneration(owner, nil, 2, "rev2", "owner", now); err != nil {
		t.Fatal(err)
	}
	versionNo := 1
	generation := int64(2)
	event := publicationEventRecord{ReportID: "current", OwnerID: "owner", Operation: "publish",
		VersionNo: &versionNo, GenerationNo: &generation, Status: "succeeded",
		RequestedBy: "owner", OccurredAt: now}
	if _, err := db.Exec(`CREATE TRIGGER reject_report_activation BEFORE UPDATE OF status ON reports
		WHEN OLD.id='current' AND NEW.status='active'
		BEGIN SELECT RAISE(ABORT,'report activation failed'); END`); err != nil {
		t.Fatal(err)
	}
	if err := transport.activatePublication(owner, "current", versionNo, generation, now, event); err == nil {
		t.Fatal("report activation failure must abort the transaction")
	}
	var pendingStatus string
	if err := db.QueryRowContext(ctx, `SELECT status FROM runtime_generations WHERE generation_no=2`).Scan(&pendingStatus); err != nil || pendingStatus != "building" {
		t.Fatalf("generation after rollback=%q err=%v", pendingStatus, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT publication_status FROM report_publications WHERE report_id='current'`).Scan(&pendingStatus); err != nil || pendingStatus != "pending" {
		t.Fatalf("publication after rollback=%q err=%v", pendingStatus, err)
	}
	if _, err := db.Exec(`DROP TRIGGER reject_report_activation`); err != nil {
		t.Fatal(err)
	}
	if err := transport.activatePublication(owner, "current", versionNo, generation, now, event); err != nil {
		t.Fatal(err)
	}
	var currentStatus string
	var currentETag int64
	if err := db.QueryRowContext(ctx, `SELECT status,etag FROM reports WHERE id='current'`).Scan(&currentStatus, &currentETag); err != nil || currentStatus != "active" || currentETag != 2 {
		t.Fatalf("activated report status=%q etag=%d err=%v", currentStatus, currentETag, err)
	}
	for _, id := range []string{"current", "other"} {
		snapshot, found, readErr := transport.publicationSnapshot(owner, nil, id)
		if readErr != nil || !found || snapshot.status != "active" ||
			!snapshot.activeGeneration.Valid || snapshot.activeGeneration.Int64 != 2 {
			t.Fatalf("committed %s=%+v found=%v err=%v", id, snapshot, found, readErr)
		}
	}
	var reportCount int
	if err := db.QueryRowContext(ctx, `SELECT report_count FROM runtime_generations WHERE generation_no=2`).Scan(&reportCount); err != nil || reportCount != 2 {
		t.Fatalf("active generation report count=%d err=%v", reportCount, err)
	}
	var newStatus, oldStatus string
	var retiredAt sql.NullTime
	if err := db.QueryRowContext(ctx, `SELECT status FROM runtime_generations WHERE generation_no=2`).Scan(&newStatus); err != nil || newStatus != "active" {
		t.Fatalf("new generation status=%q err=%v", newStatus, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT status,retired_at FROM runtime_generations WHERE generation_no=1`).Scan(&oldStatus, &retiredAt); err != nil || oldStatus != "retired" || !retiredAt.Valid {
		t.Fatalf("old generation status=%q retired=%v err=%v", oldStatus, retiredAt, err)
	}
	if err := transport.insertBuildingGeneration(owner, nil, 3, "unpublish:current:3", "owner", now); err != nil {
		t.Fatal(err)
	}
	stageTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	expectedStage := int64(2)
	if err := transport.writePublicationStage(owner, stageTx, "unpublish", 3, &publicationstage.StoredPublication{
		ReportId: "current", DesiredGeneration: &expectedStage, PublicationStatus: "unpublishing",
		Has: &publicationstage.StoredPublicationHas{ReportId: true, DesiredVersionNo: true,
			DesiredGeneration: true, PublicationStatus: true, FailureJson: true},
	}); err != nil {
		stageTx.Rollback()
		t.Fatal(err)
	}
	if err := stageTx.Commit(); err != nil {
		t.Fatal(err)
	}
	unpublishGeneration := int64(3)
	unpublishEvent := publicationEventRecord{ReportID: "current", OwnerID: "owner", Operation: "unpublish",
		VersionNo: &versionNo, GenerationNo: &unpublishGeneration, Status: "succeeded",
		RequestedBy: "owner", OccurredAt: now}
	if _, err := db.Exec(`CREATE TRIGGER reject_report_disable BEFORE UPDATE OF status ON reports
		WHEN OLD.id='current' AND NEW.status='disabled'
		BEGIN SELECT RAISE(ABORT,'report disable failed'); END`); err != nil {
		t.Fatal(err)
	}
	if err := transport.activateUnpublish(owner, "current", unpublishGeneration, now, unpublishEvent); err == nil {
		t.Fatal("report disable failure must abort unpublish activation")
	}
	staged, found, err := transport.publicationSnapshot(owner, nil, "current")
	if err != nil || !found || staged.status != "unpublishing" || staged.desiredGeneration != 3 {
		t.Fatalf("unpublish rollback publication=%+v found=%v err=%v", staged, found, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT status FROM runtime_generations WHERE generation_no=3`).Scan(&pendingStatus); err != nil || pendingStatus != "building" {
		t.Fatalf("unpublish rollback generation=%q err=%v", pendingStatus, err)
	}
	if _, err := db.Exec(`DROP TRIGGER reject_report_disable`); err != nil {
		t.Fatal(err)
	}
	if err := transport.activateUnpublish(owner, "current", unpublishGeneration, now, unpublishEvent); err != nil {
		t.Fatal(err)
	}
	if _, found, err := transport.publicationSnapshot(owner, nil, "current"); err != nil || found {
		t.Fatalf("unpublished current still present=%v err=%v", found, err)
	}
	other, found, err = transport.publicationSnapshot(owner, nil, "other")
	if err != nil || !found || other.status != "active" || other.activeGeneration.Int64 != 3 {
		t.Fatalf("other after unpublish=%+v found=%v err=%v", other, found, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT report_count,status FROM runtime_generations WHERE generation_no=3`).Scan(&reportCount, &newStatus); err != nil || reportCount != 1 || newStatus != "active" {
		t.Fatalf("unpublish generation count=%d status=%q err=%v", reportCount, newStatus, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT status,etag FROM reports WHERE id='current'`).Scan(&currentStatus, &currentETag); err != nil || currentStatus != "disabled" || currentETag != 3 {
		t.Fatalf("unpublished report status=%q etag=%d err=%v", currentStatus, currentETag, err)
	}
	var versionState string
	if err := db.QueryRowContext(ctx, `SELECT state FROM report_versions WHERE report_id='current' AND version_no=1`).Scan(&versionState); err != nil || versionState != "superseded" {
		t.Fatalf("unpublished version state=%q err=%v", versionState, err)
	}
}
