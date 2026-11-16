package sqltransport

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/viant/datly-studio/sdk"
)

type deadlineWarmup struct {
	reached chan time.Time
	partial bool
}

func (w deadlineWarmup) Warmup(ctx context.Context, _ string, _ int) (*sdk.WarmupResult, error) {
	deadline, _ := ctx.Deadline()
	w.reached <- deadline
	<-ctx.Done()
	if w.partial {
		return &sdk.WarmupResult{CompletedCases: 1, Entries: 2}, ctx.Err()
	}
	return nil, ctx.Err()
}

func TestWarmupAcceptedDeadlineAndTerminalStatus(t *testing.T) {
	for _, tc := range []struct {
		name     string
		duration *string
		budget   time.Duration
		partial  bool
	}{
		{"default", nil, 5 * time.Minute, false},
		{"longer", warmupDuration("10m"), 10 * time.Minute, false},
		{"timeout", warmupDuration("300ms"), 300 * time.Millisecond, false},
		{"partial-timeout", warmupDuration("300ms"), 300 * time.Millisecond, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newImportFixture(t)
			f.transport.Now = nil // Deadlines use the same real clock as worker contexts.
			v, err := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
			if err != nil {
				t.Fatal(err)
			}
			reached := make(chan time.Time, 1)
			if tc.budget >= time.Minute {
				// Observe the long deadline without waiting for it to expire.
				f.transport.Warmup = deadlineCapture{reached: reached}
			} else {
				f.transport.Warmup = deadlineWarmup{reached: reached, partial: tc.partial}
			}
			f.transport.WarmupOperationTimeout = tc.duration
			caller, cancel := context.WithCancel(f.ctx)
			run, err := f.client.Versions().Warmup(caller, f.report.ID, v.VersionNo)
			if err != nil {
				cancel()
				t.Fatal(err)
			}
			cancel() // Accepted work must survive browser cancellation.
			var persisted time.Time
			if err = f.db.QueryRowContext(f.ctx, "SELECT deadline_at FROM component_warmup_runs WHERE run_id=?", run.RunID).Scan(&persisted); err != nil {
				t.Fatal(err)
			}
			if actual := persisted.Sub(run.RequestedAt); actual != tc.budget {
				t.Fatalf("budget=%v want=%v", actual, tc.budget)
			}
			select {
			case actual := <-reached:
				if !actual.Equal(persisted) {
					t.Fatalf("worker deadline=%v persisted=%v", actual, persisted)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("worker never started")
			}
			limit := time.Now().Add(3 * time.Second)
			var completed *sdk.WarmupRun
			for time.Now().Before(limit) {
				completed, err = f.client.Versions().WarmupRun(f.ctx, f.report.ID, run.RunID)
				if err == nil && completed.CompletedAt != nil {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			want := "completed"
			if tc.budget < time.Minute {
				want = "failed"
				if tc.partial {
					want = "partial"
				}
			}
			if err != nil || completed == nil || completed.Status != want || completed.CompletedAt == nil {
				t.Fatalf("terminal=%+v err=%v", completed, err)
			}
			if want != "completed" && (len(completed.Diagnostics) != 1 || completed.Diagnostics[0].Code != "warmup_timeout") {
				t.Fatalf("timeout diagnostic=%+v", completed.Diagnostics)
			}
			var key sql.NullString
			if err = f.db.QueryRowContext(f.ctx, "SELECT active_key FROM component_warmup_runs WHERE run_id=?", run.RunID).Scan(&key); err != nil || key.Valid {
				t.Fatalf("terminal active key retained: %v", err)
			}
		})
	}
}

type deadlineCapture struct{ reached chan time.Time }

func (w deadlineCapture) Warmup(ctx context.Context, _ string, _ int) (*sdk.WarmupResult, error) {
	deadline, _ := ctx.Deadline()
	w.reached <- deadline
	return &sdk.WarmupResult{}, nil
}
func warmupDuration(value string) *string { return &value }

func TestRecoveryUsesAcceptedDeadlineNotHostTimeout(t *testing.T) {
	f := newImportFixture(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	for _, tc := range []struct {
		id        string
		requested time.Time
		deadline  any
		want      string
	}{
		{"long-running", now.Add(-6 * time.Minute), now.Add(4 * time.Minute), "running"},
		{"expired-short", now.Add(-time.Second), now.Add(-time.Millisecond), "failed"},
		{"legacy-old", now.Add(-6 * time.Minute), nil, "failed"},
		{"legacy-recent", now.Add(-time.Minute), nil, "running"},
		{"exact-boundary", now.Add(-time.Minute), now, "failed"},
	} {
		_, err := f.db.ExecContext(f.ctx, `INSERT INTO component_warmup_runs(run_id,report_id,version_no,source_revision,spec_hash,plan_key,active_key,status,requested_by,target_json,requested_at,deadline_at,updated_at,updated_by) VALUES(?,?,1,1,'hash','plan',?,'running','owner','{}',?,?,?,'owner')`, tc.id, f.report.ID, tc.id, tc.requested, tc.deadline, tc.requested)
		if err != nil {
			t.Fatal(err)
		}
	}
	// Recovery must be independent of this host's newly configured budget.
	f.transport.WarmupOperationTimeout = warmupDuration("1ns")
	if err := f.transport.recoverExpiredWarmupRuns(f.ctx, now); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]string{"long-running": "running", "expired-short": "failed", "legacy-old": "failed", "legacy-recent": "running", "exact-boundary": "failed"} {
		var actual string
		if err := f.db.QueryRowContext(f.ctx, "SELECT status FROM component_warmup_runs WHERE run_id=?", id).Scan(&actual); err != nil || actual != want {
			t.Fatalf("%s status=%s want=%s err=%v", id, actual, want, err)
		}
	}
}
