package sqltransport

import (
	"context"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/viant/datly-studio/sdk"
)

type terminalDiagnosticLog chan string

func (w terminalDiagnosticLog) Write(p []byte) (int, error) {
	w <- string(p)
	return len(p), nil
}

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
			messages := make(terminalDiagnosticLog, 8)
			previous := log.Writer()
			log.SetOutput(messages)
			t.Cleanup(func() { log.SetOutput(previous) })
			f.transport.Warmup = terminalDiagnosticWarmup{status: status}
			run, err := f.client.Versions().Warmup(f.ctx, f.report.ID, v.VersionNo)
			if err != nil {
				t.Fatal(err)
			}
			select {
			case message := <-messages:
				for _, part := range []string{"Studio warmup terminal persistence failed:", run.RunID, f.report.ID, "version_no=1", `intended_status="` + status + `"`, "error_type="} {
					if !strings.Contains(message, part) {
						t.Fatalf("missing %q in diagnostic %q", part, message)
					}
				}
				for _, secret := range []string{"private-storage-canary", "fixture-secret", "private execution error", "CREATE TRIGGER"} {
					if strings.Contains(message, secret) {
						t.Fatalf("diagnostic exposes private error detail: %q", message)
					}
				}
			case <-time.After(3 * time.Second):
				t.Fatal("terminal write failure was not logged")
			}
			persisted, err := f.client.Versions().WarmupRun(f.ctx, f.report.ID, run.RunID)
			if err != nil || persisted.Status != "running" || persisted.CompletedAt != nil {
				t.Fatalf("failed write must not fabricate a terminal record: %+v err=%v", persisted, err)
			}
		})
	}
}
