package sqltransport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	storedreader "github.com/viant/datly-studio/studio/report_warmup_runs/store_read"
	storedwriter "github.com/viant/datly-studio/studio/report_warmup_runs/store_write"
	xhandler "github.com/viant/xdatly/handler"
	_ "modernc.org/sqlite"
)

func TestWarmupRunStoreReadFiltersOrderAndNulls(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	for _, item := range []struct {
		id, report string
		version    int
		at         time.Time
		active     any
	}{
		{"a", "r1", 1, now.Add(-time.Minute), nil},
		{"b", "r1", 1, now, "r1:1:plan"},
		{"c", "r1", 1, now, nil},
		{"d", "r1", 2, now.Add(time.Hour), nil},
		{"e", "r2", 1, now.Add(time.Hour), nil},
	} {
		_, err = db.ExecContext(ctx, `INSERT INTO report_warmup_runs(run_id,report_id,version_no,source_revision,spec_hash,plan_key,active_key,status,requested_by,target_json,requested_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, item.id, item.report, item.version, 3, "hash", "plan", item.active, "accepted", "owner", `{}`, item.at)
		if err != nil {
			t.Fatal(err)
		}
	}
	transport := &Transport{DB: db}
	_, err = db.ExecContext(ctx, `UPDATE report_warmup_runs SET started_at=?,completed_at=?,max_cases=7,row_limit=9,duration_ns=123,target_json=?,diagnostics_json=? WHERE run_id='b'`, now, now, `{"view":"reader","cacheName":"cache"}`, `[{"severity":"error","code":"warmup_failed","message":"failed"}]`)
	if err != nil {
		t.Fatal(err)
	}
	page := &sdk.WarmupRunPage{}
	if err := transport.listWarmupRuns(ctx, warmupRunListRequest{ReportID: "r1", VersionNo: 1, Input: sdk.ListWarmupRunsInput{Limit: 2, Offset: 1}}, page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.Items[0].RunID != "b" || page.Items[1].RunID != "a" || page.Limit != 2 || page.Offset != 1 {
		t.Fatalf("page=%+v", page)
	}
	if page.Items[1].StartedAt != nil || page.Items[1].CompletedAt != nil || page.Items[1].MaxCases != nil || page.Items[1].RowLimit != nil || len(page.Items[1].Diagnostics) != 0 {
		t.Fatalf("null fields=%+v", page.Items[1])
	}
	if page.Items[0].StartedAt == nil || page.Items[0].CompletedAt == nil || page.Items[0].MaxCases == nil || *page.Items[0].MaxCases != 7 || page.Items[0].RowLimit == nil || *page.Items[0].RowLimit != 9 || page.Items[0].Duration != 123 || page.Items[0].Target.CacheName != "cache" || len(page.Items[0].Diagnostics) != 1 {
		t.Fatalf("populated fields=%+v", page.Items[0])
	}
	run, err := transport.readWarmupRun(ctx, "r1", "b")
	if err != nil || run.RunID != "b" {
		t.Fatalf("run=%+v err=%v", run, err)
	}
	active, err := transport.readWarmupRows(ctx, &storedreader.Input{ActiveKey: "r1:1:plan", PageLimit: 2})
	if err != nil || len(active) != 1 || active[0].RunID != "b" {
		t.Fatalf("active=%+v err=%v", active, err)
	}
	if _, err := transport.readWarmupRun(ctx, "r2", "b"); err == nil {
		t.Fatal("expected report-scoped not found")
	} else {
		var sdkErr *sdk.Error
		if !errors.As(err, &sdkErr) || sdkErr.Code != sdk.ErrorNotFound {
			t.Fatalf("err=%v", err)
		}
	}
}

func TestRecoverExpiredWarmupRunsUsesNativeMatchedWrites(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	for i := 0; i < 105; i++ {
		status := "accepted"
		if i%2 == 1 {
			status = "running"
		}
		id := fmt.Sprintf("old-%03d", i)
		if _, err := db.ExecContext(ctx, `INSERT INTO report_warmup_runs
			(run_id,report_id,version_no,source_revision,spec_hash,plan_key,active_key,status,requested_by,target_json,requested_at,created_at,created_by,updated_at,updated_by)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, id, "r1", 1, 1, "hash", "plan", id, status, "owner", "{}",
			now.Add(-warmupRunTimeout-time.Minute), now.Add(-warmupRunTimeout-time.Minute), "owner",
			now.Add(-warmupRunTimeout-time.Minute), "owner"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO report_warmup_runs
		(run_id,report_id,version_no,source_revision,spec_hash,plan_key,active_key,status,requested_by,target_json,requested_at,created_at,created_by,updated_at,updated_by)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, "recent", "r1", 1, 1, "hash", "plan", "recent", "accepted", "owner", "{}", now, now, "owner", now, "owner"); err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db}
	if err := transport.recoverExpiredWarmupRuns(ctx, now); err != nil {
		t.Fatal(err)
	}
	var recovered int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM report_warmup_runs
		WHERE status='failed' AND active_key IS NULL AND completed_at IS NOT NULL
		AND updated_by='system:datly-studio' AND diagnostics_json LIKE '%warmup_expired%'`).Scan(&recovered); err != nil || recovered != 105 {
		t.Fatalf("recovered=%d err=%v", recovered, err)
	}
	var recentStatus string
	if err := db.QueryRowContext(ctx, `SELECT status FROM report_warmup_runs WHERE run_id='recent'`).Scan(&recentStatus); err != nil || recentStatus != "accepted" {
		t.Fatalf("recent status=%q err=%v", recentStatus, err)
	}
	if err := transport.recoverExpiredWarmupRuns(ctx, now); err != nil {
		t.Fatalf("idempotent recovery: %v", err)
	}
}

func TestTranscribedWarmupWriterRejectsStaleUpdatedAt(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	transport := &Transport{DB: db}
	now := time.Now().UTC().Truncate(time.Microsecond)
	actor, active := "alice", "active-r1"
	if err := transport.writeWarmupRun(ctx, &storedwriter.StoredWarmupRun{
		RunId: "r1", ReportId: "report", VersionNo: 1, SourceRevision: 1,
		SpecHash: "hash", PlanKey: "plan", ActiveKey: &active, Status: "accepted",
		RequestedBy: actor, TargetJson: json.RawMessage(`{}`), RequestedAt: now,
		CreatedAt: &now, CreatedBy: &actor, UpdatedAt: &now, UpdatedBy: &actor,
		Has: &storedwriter.StoredWarmupRunHas{RunId: true, ReportId: true, VersionNo: true,
			SourceRevision: true, SpecHash: true, PlanKey: true, ActiveKey: true,
			Status: true, RequestedBy: true, TargetJson: true, RequestedAt: true,
			CreatedAt: true, CreatedBy: true, UpdatedAt: true, UpdatedBy: true}}); err != nil {
		t.Fatal(err)
	}
	before, err := transport.readWarmupRun(ctx, "report", "r1")
	if err != nil || before.UpdatedAt == nil {
		t.Fatalf("created run=%+v err=%v", before, err)
	}
	started := now.Add(time.Second)
	if err := transport.writeWarmupRun(ctx, &storedwriter.StoredWarmupRun{RunId: "r1", Status: "running",
		UpdatedAt: before.UpdatedAt, UpdatedBy: &actor, StartedAt: &started,
		Has: &storedwriter.StoredWarmupRunHas{RunId: true, Status: true,
			UpdatedAt: true, UpdatedBy: true, StartedAt: true}}); err != nil {
		t.Fatal(err)
	}
	worker := "system:datly-studio"
	err = transport.writeWarmupRun(ctx, &storedwriter.StoredWarmupRun{RunId: "r1", Status: "failed",
		UpdatedAt: before.UpdatedAt, UpdatedBy: &worker,
		Has: &storedwriter.StoredWarmupRunHas{RunId: true, Status: true, UpdatedAt: true, UpdatedBy: true}})
	var conflict *xhandler.Conflict
	if !errors.As(err, &conflict) {
		t.Fatalf("stale timestamp did not conflict: %v", err)
	}
	after, err := transport.readWarmupRun(ctx, "report", "r1")
	if err != nil || after.Status != "running" || after.UpdatedAt == nil || !after.UpdatedAt.After(*before.UpdatedAt) {
		t.Fatalf("stale update changed run=%+v err=%v", after, err)
	}
}
