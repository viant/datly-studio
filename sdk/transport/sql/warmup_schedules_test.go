package sqltransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/viant/datly-studio/sdk"
	runwrite "github.com/viant/datly-studio/studio/report_warmup_runs/store_write"
	"testing"
	"time"
)

func TestWarmupScheduleManagementAndAtomicAdmission(t *testing.T) {
	f := newImportFixtureDSN(t, "?_txlock=immediate&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	v, e := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
	if e != nil {
		t.Fatal(e)
	}
	initial, e := f.client.Versions().GetWarmupSchedule(f.ctx, f.report.ID, v.VersionNo)
	if e != nil || initial.Schedule != nil {
		t.Fatalf("empty schedule: %+v %v", initial, e)
	}
	input := sdk.SaveWarmupScheduleInput{IntervalSeconds: 3600, FirstRunAt: f.now.Add(-3 * time.Hour), Enabled: true}
	saved, e := f.client.Versions().SaveWarmupSchedule(f.ctx, f.report.ID, v.VersionNo, input)
	if e != nil {
		t.Fatal(e)
	}
	s := saved.Schedule
	if s == nil || s.Revision != 1 || s.CreatedBy != "owner" {
		t.Fatalf("created %+v", s)
	}
	if _, e = f.client.Versions().SaveWarmupSchedule(f.ctx, f.report.ID, v.VersionNo, input); e == nil {
		t.Fatal("stale create accepted")
	}
	read, e := f.client.Versions().GetWarmupSchedule(f.ctx, f.report.ID, v.VersionNo)
	if e != nil || read.Schedule.ScheduleID != s.ScheduleID {
		t.Fatalf("readback %+v %v", read, e)
	}
	now := f.now
	deadline := now.Add(5 * time.Minute)
	actor := "owner"
	active := "proof-active"
	row := &runwrite.StoredWarmupRun{NamespaceId: s.NamespaceID, RunId: "schedule-run", ReportId: f.report.ID, VersionNo: v.VersionNo, SourceRevision: v.SourceRevision, SpecHash: v.SpecHash, PlanKey: "plan", ActiveKey: &active, Status: "accepted", RequestedBy: actor, TargetJson: json.RawMessage(`{}`), RequestedAt: now, DeadlineAt: &deadline, CreatedAt: &now, CreatedBy: &actor, UpdatedAt: &now, UpdatedBy: &actor}
	// A child constraint fails after the parent update: both changes must roll back.
	_, e = f.db.ExecContext(f.ctx, `CREATE TRIGGER fail_schedule_run BEFORE INSERT ON component_warmup_runs WHEN NEW.run_id='schedule-run' BEGIN SELECT RAISE(ABORT,'fixture insert failure'); END`)
	if e != nil {
		t.Fatal(e)
	}
	if e = f.transport.consumeWarmupSchedule(f.ctx, s, row, row.RunId, now); e == nil {
		t.Fatal("child failure was ignored")
	}
	after, e := f.client.Versions().GetWarmupSchedule(f.ctx, f.report.ID, v.VersionNo)
	if e != nil || after.Schedule.Revision != 1 || !after.Schedule.NextDueAt.Equal(s.NextDueAt) {
		t.Fatalf("partial schedule update %+v %v", after, e)
	}
	if _, e = f.db.ExecContext(f.ctx, "DROP TRIGGER fail_schedule_run"); e != nil {
		t.Fatal(e)
	}
	if e = f.transport.consumeWarmupSchedule(f.ctx, s, row, row.RunId, now); e != nil {
		t.Fatal(e)
	}
	after, e = f.client.Versions().GetWarmupSchedule(f.ctx, f.report.ID, v.VersionNo)
	if e != nil {
		t.Fatal(e)
	}
	if after.Schedule.Revision != 2 || after.Schedule.LastRunID == nil || *after.Schedule.LastRunID != row.RunId || !after.Schedule.NextDueAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("consumed %+v", after.Schedule)
	}
	if e = f.transport.consumeWarmupSchedule(f.ctx, s, nil, row.RunId, now); e == nil {
		t.Fatal("same occurrence consumed twice")
	}
	// Disabling changes the token and fences an already inspected occurrence.
	input.ExpectedRevision = 2
	input.Enabled = false
	disabled, e := f.client.Versions().SaveWarmupSchedule(f.ctx, f.report.ID, v.VersionNo, input)
	if e != nil || disabled.Schedule.Revision != 3 {
		t.Fatalf("disable %+v %v", disabled, e)
	}
	if e = f.transport.consumeWarmupSchedule(f.ctx, after.Schedule, nil, row.RunId, now.Add(time.Hour)); e == nil {
		t.Fatal("disabled schedule consumed")
	}
	if _, e = f.client.Versions().DeleteWarmupSchedule(f.ctx, f.report.ID, v.VersionNo, sdk.DeleteWarmupScheduleInput{ExpectedRevision: 2}); e == nil {
		t.Fatal("stale delete accepted")
	}
	if _, e = f.client.Versions().DeleteWarmupSchedule(f.ctx, f.report.ID, v.VersionNo, sdk.DeleteWarmupScheduleInput{ExpectedRevision: 3}); e != nil {
		t.Fatal(e)
	}
	final, e := f.client.Versions().GetWarmupSchedule(f.ctx, f.report.ID, v.VersionNo)
	if e != nil || final.Schedule != nil {
		t.Fatalf("deleted %+v %v", final, e)
	}
	if _, e = f.client.Versions().WarmupRun(f.ctx, f.report.ID, row.RunId); e != nil {
		t.Fatalf("schedule removal deleted run history: %v", e)
	}
}
func TestWarmupScheduleInvalidConfiguration(t *testing.T) {
	f := newImportFixture(t)
	v, e := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
	if e != nil {
		t.Fatal(e)
	}
	for _, input := range []sdk.SaveWarmupScheduleInput{{IntervalSeconds: 0, FirstRunAt: f.now}, {IntervalSeconds: -1, FirstRunAt: f.now}, {IntervalSeconds: 1}, {IntervalSeconds: 1 << 62, FirstRunAt: f.now}} {
		_, e = f.client.Versions().SaveWarmupSchedule(f.ctx, f.report.ID, v.VersionNo, input)
		var se *sdk.Error
		if !errors.As(e, &se) || se.Code != sdk.ErrorInvalidArgument {
			t.Fatalf("input=%+v err=%v", input, e)
		}
	}
	if e = f.transport.RunDueWarmupSchedules(context.Background()); e == nil {
		t.Fatal("unverified scheduler admitted")
	}
}

func TestWarmupScheduleCoversExistingActiveRun(t *testing.T) {
	f := newImportFixtureDSN(t, "?_txlock=immediate&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	f.transport.Now = nil
	v, e := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
	if e != nil {
		t.Fatal(e)
	}
	saved, e := f.client.Versions().SaveWarmupSchedule(f.ctx, f.report.ID, v.VersionNo, sdk.SaveWarmupScheduleInput{IntervalSeconds: 3600, FirstRunAt: time.Now().UTC().Add(-time.Minute), Enabled: true})
	if e != nil {
		t.Fatal(e)
	}
	release := make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	f.transport.Warmup = warmupStub{wait: release, result: &sdk.WarmupResult{CompletedCases: 1, Entries: 1}}
	manual, e := f.client.Versions().Warmup(f.ctx, f.report.ID, v.VersionNo)
	if e != nil {
		t.Fatal(e)
	}
	var covered sdk.WarmupRun
	if e = f.transport.startWarmupRun(f.ctx, versionIdentityRequest{ReportID: f.report.ID, VersionNo: v.VersionNo}, &covered, saved.Schedule); e != nil {
		t.Fatal(e)
	}
	if covered.RunID != manual.RunID {
		t.Fatal("created a second run instead of coalescing")
	}
	after, e := f.client.Versions().GetWarmupSchedule(f.ctx, f.report.ID, v.VersionNo)
	if e != nil || after.Schedule.Revision != 2 || after.Schedule.LastRunID == nil || *after.Schedule.LastRunID != manual.RunID {
		t.Fatalf("coalesced schedule %+v %v", after, e)
	}
	var count int
	if e = f.db.QueryRowContext(f.ctx, "SELECT COUNT(*) FROM component_warmup_runs WHERE report_id=?", f.report.ID).Scan(&count); e != nil || count != 1 {
		t.Fatalf("runs=%d err=%v", count, e)
	}
	// Release the existing worker and observe terminal persistence before fixture teardown.
	close(release)
	until := time.Now().Add(3 * time.Second)
	for time.Now().Before(until) {
		run, e := f.client.Versions().WarmupRun(f.ctx, f.report.ID, manual.RunID)
		if e == nil && run.CompletedAt != nil {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("coalesced worker did not finish")
}

func TestWarmupScheduleDueReaderPagesPastFailedCandidates(t *testing.T) {
	f := newImportFixture(t)
	v, e := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
	if e != nil {
		t.Fatal(e)
	}
	// Seed independent numbered targets directly; assertions exercise the native reader.
	for i := 1; i <= 101; i++ {
		if i != v.VersionNo {
			if _, e = f.db.ExecContext(f.ctx, `INSERT INTO component_versions(report_id,version_no,state,authoring_mode,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at) SELECT report_id,?,state,authoring_mode,component_spec_json,spec_format_version,?,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at FROM component_versions WHERE report_id=? AND version_no=?`, i, fmt.Sprintf("%064x", i), f.report.ID, v.VersionNo); e != nil {
				t.Fatal(e)
			}
		}
		ns, e := f.transport.resourceWorkspaceID(f.ctx, nil, f.report.ID)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = f.db.ExecContext(f.ctx, `INSERT INTO component_warmup_schedules(schedule_id,namespace_id,report_id,version_no,interval_seconds,first_run_at,next_due_at,enabled,revision,created_at,created_by,updated_at,updated_by) VALUES(?,?,?,?,3600,?,?,1,1,?,'owner',?,'owner')`, fmt.Sprintf("schedule-%03d", i), ns, f.report.ID, i, f.now, f.now, f.now, f.now); e != nil {
			t.Fatal(e)
		}
	}
	rows, e := f.transport.readWarmupSchedules(f.ctx, "", 0, &f.now)
	if e != nil || len(rows) != 101 || rows[100].ScheduleID != "schedule-101" {
		t.Fatalf("due pages=%d err=%v", len(rows), e)
	}
}
