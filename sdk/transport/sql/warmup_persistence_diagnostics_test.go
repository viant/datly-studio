package sqltransport

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/viant/datly-studio/internal/logtest"
	"strings"
	"testing"
	"time"

	"github.com/viant/datly-studio/sdk"
)

type terminalDiagnosticWarmup struct{ status string }

func (w terminalDiagnosticWarmup) Warmup(ctx context.Context, _ string, _ int) (*sdk.WarmupResult, error) {
	if w.status == "completed" {
		return &sdk.WarmupResult{}, nil
	}
	if w.status == "partial" {
		return &sdk.WarmupResult{CompletedCases: 1}, context.DeadlineExceeded
	}
	return nil, errors.New("private execution error")
}

func TestWarmupTerminalPersistenceDiagnostic(t *testing.T) {
	for _, status := range []string{"completed", "failed", "partial"} {
		t.Run(status, func(t *testing.T) {
			f := newImportFixtureDSN(t, "?_txlock=immediate&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
			f.transport.Now = nil
			v, err := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
			if err != nil {
				t.Fatal(err)
			}
			// Fail the actual generated writer, after native execution, without
			// replacing persistence or changing its transaction behavior.
			_, err = f.db.ExecContext(f.ctx, `CREATE TRIGGER fail_terminal BEFORE UPDATE ON component_warmup_runs WHEN NEW.status IN ('completed','failed','partial') BEGIN SELECT RAISE(ABORT, 'private-storage-canary password=fixture-secret'); END`)
			if err != nil {
				t.Fatal(err)
			}
			logs := logtest.CaptureLogs(t)
			f.transport.Warmup = terminalDiagnosticWarmup{status: status}
			run, err := f.client.Versions().Warmup(f.ctx, f.report.ID, v.VersionNo)
			if err != nil {
				t.Fatal(err)
			}

			waitCtx, cancel := context.WithTimeout(f.ctx, 3*time.Second)
			defer cancel()
			event := logs.Wait(waitCtx, t, "Studio warmup terminal persistence failed")
			if event["run_id"] != run.RunID || event["component_id"] != f.report.ID || event["version_no"] != float64(1) || event["intended_status"] != status || event["stage"] != "terminal_persistence" || event["transaction_state"] != "rolled_back" {
				t.Fatalf("persistence event: %+v", event)
			}
			execution := logs.Wait(waitCtx, t, "Studio warmup execution finished")
			if execution["run_id"] != run.RunID || execution["outcome"] != status || execution["timed_out"] != (status == "partial") {
				t.Fatalf("execution event: %+v", execution)
			}
			data, _ := json.Marshal(logs.Entries())
			for _, secret := range []string{"private-storage-canary", "fixture-secret", "private execution error", "CREATE TRIGGER"} {
				if strings.Contains(string(data), secret) {
					t.Fatalf("diagnostic exposes private detail: %s", data)
				}
			}

			persisted, err := f.client.Versions().WarmupRun(f.ctx, f.report.ID, run.RunID)
			if err != nil || persisted.Status != "running" || persisted.CompletedAt != nil {
				t.Fatalf("failed write must not fabricate a terminal record: %+v err=%v", persisted, err)
			}
		})
	}
}

func TestWarmupStartupReadFailureDiagnostic(t *testing.T) {
	f := newImportFixture(t)
	version, err := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
	if err != nil {
		t.Fatal(err)
	}
	// The business operation must not run if its committed running record cannot
	// be read. This fixture fault exercises the actual native transition/read.
	if _, err = f.db.ExecContext(f.ctx, `CREATE TRIGGER remove_running AFTER UPDATE ON component_warmup_runs WHEN NEW.status='running' BEGIN DELETE FROM component_warmup_runs WHERE run_id=NEW.run_id; END`); err != nil {
		t.Fatal(err)
	}
	logs := logtest.CaptureLogs(t)
	warmup := &countedWarmup{status: "completed"}
	f.transport.Warmup = warmup
	run, err := f.client.Versions().Warmup(f.ctx, f.report.ID, version.VersionNo)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(f.ctx, 3*time.Second)
	defer cancel()
	event := logs.Wait(ctx, t, "Studio warmup worker startup failed")
	if event["run_id"] != run.RunID || event["stage"] != "worker_start_read" || warmup.calls != 0 {
		t.Fatalf("startup failure: %+v calls=%d", event, warmup.calls)
	}
	for _, entry := range logs.Entries() {
		if entry["msg"] == "Studio warmup execution finished" || entry["msg"] == "Studio warmup terminal persistence failed" {
			t.Fatalf("startup incorrectly reported as execution/persistence completion: %+v", entry)
		}
	}
}
