package sqltransport

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	publicationrecover "github.com/viant/datly-studio/studio/report_publications/store_recover"
	xhandler "github.com/viant/xdatly/handler"
	_ "modernc.org/sqlite"
)

func TestExpiredStageRecoveryUsesMatchedDatlyComponents(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	stale := now.Add(-stagedGenerationLease - time.Minute)
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	mustExec(`INSERT INTO connectors(name,driver,owner_id,status,etag,created_at,updated_at) VALUES('main','sqlite','owner','active',1,?,?)`, now, now)
	mustExec(`INSERT INTO namespaces(owner_id,name,title,status,etag,created_at,updated_at) VALUES('owner','general','General','active',1,?,?)`, now, now)
	for _, id := range []string{"restage", "unpublish", "first"} {
		mustExec(`INSERT INTO reports(id,namespace,slug,title,owner_id,status,default_connector_name,component_scope,component_name,etag,created_at,updated_at)
			VALUES(?,'general',?,?, 'owner','active','main','example.com/reader',?,1,?,?)`, id, id, id, id, now, now)
		mustExec(`INSERT INTO report_versions(report_id,version_no,state,authoring_mode,authored_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at,published_at)
			VALUES(?,1,'published','dql','SELECT 1','{}','1',?,'{}','valid','v1','v1',1,'owner',?,?)`, id, "spec-"+id, now, now)
	}
	mustExec(`INSERT INTO runtime_generations(generation_no,source_revision,status,report_count,build_manifest_json,requested_by,requested_at,activated_at)
		VALUES(1,'rev-active','active',2,'{}','owner',?,?)`, now, now)
	for generation := int64(2); generation <= 5; generation++ {
		requestedAt := stale
		if generation == 5 {
			requestedAt = now
		}
		mustExec(`INSERT INTO runtime_generations(generation_no,source_revision,status,report_count,build_manifest_json,requested_by,requested_at)
			VALUES(?,?,'building',0,'{}','owner',?)`, generation, generation, requestedAt)
	}
	for _, item := range []struct {
		id, status string
		generation int64
		active     any
		desired    any
	}{
		{id: "restage", status: "pending", generation: 2, active: int64(1), desired: 1},
		{id: "unpublish", status: "unpublishing", generation: 4, active: int64(1), desired: nil},
		{id: "first", status: "pending", generation: 3, active: nil, desired: 1},
	} {
		mustExec(`INSERT INTO report_publications(report_id,active_version_no,desired_version_no,desired_generation,active_generation,publication_status,runtime_revision,spec_hash,published_by,published_at,activated_at)
			VALUES(?,1,?,?,?,?, 'staged','staged-spec','owner',?,?)`, item.id, item.desired, item.generation, item.active, item.status, now, now)
	}
	transport := &Transport{DB: db}
	blocked, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = transport.ensureNoStagedGeneration(ctx, blocked, now)
	var conflict *sdk.Error
	if !errors.As(err, &conflict) || conflict.Code != sdk.ErrorConflict {
		t.Fatalf("live building generation error=%v", err)
	}
	if err := blocked.Rollback(); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := db.QueryRowContext(ctx, `SELECT status FROM runtime_generations WHERE generation_no=2`).Scan(&status); err != nil || status != "building" {
		t.Fatalf("blocked recovery changed generation=%q err=%v", status, err)
	}
	mustExec(`UPDATE runtime_generations SET requested_at=? WHERE generation_no=5`, stale)
	mustExec(`CREATE TRIGGER reject_last_expiry BEFORE UPDATE OF status ON runtime_generations
		WHEN OLD.generation_no=5 AND NEW.status='failed'
		BEGIN SELECT RAISE(ABORT,'last expiry failed'); END`)
	rejected, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := transport.ensureNoStagedGeneration(ctx, rejected, now); err == nil {
		t.Fatal("last generation failure must abort recovery")
	}
	if err := rejected.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT publication_status FROM report_publications WHERE report_id='restage'`).Scan(&status); err != nil || status != "pending" {
		t.Fatalf("recovery failure changed publication=%q err=%v", status, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT status FROM runtime_generations WHERE generation_no=2`).Scan(&status); err != nil || status != "building" {
		t.Fatalf("recovery failure changed generation=%q err=%v", status, err)
	}
	mustExec(`DROP TRIGGER reject_last_expiry`)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	expectedGeneration := int64(2)
	diagnostics := `[{"code":"expired"}]`
	staleRow := &publicationrecover.StoredPublication{ReportId: "restage", DesiredGeneration: &expectedGeneration,
		PublicationStatus: "active", FailureJson: &diagnostics,
		Has: &publicationrecover.StoredPublicationHas{ReportId: true, DesiredGeneration: true,
			PublicationStatus: true, FailureJson: true}}
	var staleConflict *xhandler.Conflict
	if err := transport.writePublicationRecovery(ctx, tx, "fail", staleRow); !errors.As(err, &staleConflict) {
		t.Fatalf("stale publication status error=%v", err)
	}
	if err := transport.ensureNoStagedGeneration(ctx, tx, now); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		id, status, revision, spec string
		generation                 int64
	}{
		{id: "restage", status: "active", generation: 1, revision: "rev-active", spec: "spec-restage"},
		{id: "unpublish", status: "active", generation: 1, revision: "rev-active", spec: "spec-unpublish"},
		{id: "first", status: "failed", generation: 3, revision: "staged", spec: "staged-spec"},
	} {
		var desiredVersion sql.NullInt64
		var generation int64
		var actualStatus, revision, spec, failure string
		if err := tx.QueryRowContext(ctx, `SELECT desired_version_no,desired_generation,publication_status,runtime_revision,spec_hash,failure_json FROM report_publications WHERE report_id=?`, item.id).
			Scan(&desiredVersion, &generation, &actualStatus, &revision, &spec, &failure); err != nil ||
			!desiredVersion.Valid || desiredVersion.Int64 != 1 || generation != item.generation || actualStatus != item.status ||
			revision != item.revision || spec != item.spec || !strings.Contains(failure, "staged_generation_expired") {
			t.Fatalf("recovered %s version=%v generation=%d status=%q revision=%q spec=%q failure=%q err=%v", item.id, desiredVersion, generation, actualStatus, revision, spec, failure, err)
		}
	}
	for generation := int64(2); generation <= 5; generation++ {
		var diagnostics string
		var retired sql.NullTime
		if err := tx.QueryRowContext(ctx, `SELECT status,diagnostics_json,retired_at FROM runtime_generations WHERE generation_no=?`, generation).
			Scan(&status, &diagnostics, &retired); err != nil || status != "failed" || !retired.Valid || !strings.Contains(diagnostics, "staged_generation_expired") {
			t.Fatalf("expired generation %d status=%q diagnostics=%q retired=%v err=%v", generation, status, diagnostics, retired, err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}
