package sqltransport

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly-studio/sdk"
	stored "github.com/viant/datly-studio/studio/report_warmup_runs/store_write"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type scheduledAdmissionFixture struct {
	Schedule sdk.WarmupSchedule
	Run      stored.StoredWarmupRun
	Now      time.Time
}

func TestWarmupScheduleSeparateProcesses(t *testing.T) {
	for _, driver := range []string{"sqlite", "mysql"} {
		t.Run(driver, func(t *testing.T) {
			var f *importFixture
			var dsn string
			if driver == "sqlite" {
				f = newImportFixtureDSN(t, "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
				var seq int
				var name, file string
				if e := f.db.QueryRowContext(f.ctx, "PRAGMA database_list").Scan(&seq, &name, &file); e != nil {
					t.Fatal(e)
				}
				dsn = "file:" + file + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
			} else {
				dsn = os.Getenv("STUDIO_SCHEDULE_MYSQL_DSN")
				if dsn == "" {
					t.Skip("requires empty owned STUDIO_SCHEDULE_MYSQL_DSN")
				}
				db, e := sql.Open("mysql", dsn)
				if e != nil {
					t.Fatal(e)
				}
				defer db.Close()
				ctx := sdk.WithPrincipal(t.Context(), sdk.Principal{Subject: "owner"})
				var tables int
				if e = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE()").Scan(&tables); e != nil || tables != 0 {
					t.Fatal("requires empty owned database", e)
				}
				if e = schema.Initialize(ctx, db, "mysql"); e != nil {
					t.Fatal(e)
				}
				tr := &Transport{DB: db, Authorizer: allowAuthorizer{}}
				client, e := sdk.NewClient(tr)
				if e != nil {
					t.Fatal(e)
				}
				connector, e := client.Connectors().Create(ctx, sdk.CreateConnectorInput{Name: "source", Driver: "sqlite", DSNTemplate: "file:fixture.db"})
				if e != nil {
					t.Fatal(e)
				}
				if _, e = db.ExecContext(ctx, "UPDATE connectors SET status='active' WHERE name=?", connector.Name); e != nil {
					t.Fatal(e)
				}
				report, e := client.Components().Create(ctx, sdk.CreateComponentInput{Slug: "schedule", Title: "Schedule", DefaultConnectorName: connector.Name})
				if e != nil {
					t.Fatal(e)
				}
				f = &importFixture{ctx: ctx, db: db, transport: tr, client: client, report: report, now: time.Now().UTC().Truncate(time.Microsecond)}
			}
			v, e := f.client.Versions().Create(f.ctx, f.report.ID, sdk.CreateVersionInput{AuthoringMode: "dql", AuthoredDQL: "SELECT 1"})
			if e != nil {
				t.Fatal(e)
			}
			first := f.now.Add(-time.Minute)
			saved, e := f.client.Versions().SaveWarmupSchedule(f.ctx, f.report.ID, v.VersionNo, sdk.SaveWarmupScheduleInput{IntervalSeconds: 3600, FirstRunAt: first, Enabled: true})
			if e != nil {
				t.Fatal(e)
			}
			now := f.now
			deadline := now.Add(5 * time.Minute)
			actor := "owner"
			exe, e := os.Executable()
			if e != nil {
				t.Fatal(e)
			}
			dir := t.TempDir()
			type child struct {
				cmd *exec.Cmd
				log strings.Builder
			}
			children := make([]*child, 2)
			for i := 0; i < 2; i++ {
				id := fmt.Sprintf("scheduled-race-%d", i)
				fxt := scheduledAdmissionFixture{Schedule: *saved.Schedule, Now: now, Run: stored.StoredWarmupRun{NamespaceId: saved.Schedule.NamespaceID, RunId: id, ReportId: f.report.ID, VersionNo: v.VersionNo, SourceRevision: v.SourceRevision, SpecHash: v.SpecHash, PlanKey: "plan", ActiveKey: &id, Status: "accepted", RequestedBy: actor, TargetJson: json.RawMessage(`{}`), RequestedAt: now, DeadlineAt: &deadline, CreatedAt: &now, CreatedBy: &actor, UpdatedAt: &now, UpdatedBy: &actor}}
				data, e := json.Marshal(fxt)
				if e != nil {
					t.Fatal(e)
				}
				file := filepath.Join(dir, fmt.Sprintf("%d.json", i))
				if e = os.WriteFile(file, data, 0600); e != nil {
					t.Fatal(e)
				}
				c := &child{}
				c.cmd = exec.CommandContext(t.Context(), exe, "-test.run=^TestWarmupScheduleProcessWriter$")
				c.cmd.Env = append(os.Environ(), "SCHEDULE_PROOF_DRIVER="+driver, "SCHEDULE_PROOF_DSN="+dsn, "SCHEDULE_PROOF_INPUT="+file, "SCHEDULE_PROOF_GATE="+filepath.Join(dir, "release"))
				c.cmd.Stdout = &c.log
				c.cmd.Stderr = &c.log
				if e = c.cmd.Start(); e != nil {
					t.Fatal(e)
				}
				children[i] = c
			}
			// Both processes receive the same inspected occurrence and synchronize before
			// native dispatch; they share neither a transport mutex nor a database pool.
			if e = os.WriteFile(filepath.Join(dir, "release"), nil, 0600); e != nil {
				t.Fatal(e)
			}
			wins := 0
			for _, c := range children {
				e = c.cmd.Wait()
				if e == nil {
					wins++
				} else {
					t.Log(c.log.String())
				}
			}
			if wins != 1 {
				t.Fatalf("successful independent writers=%d", wins)
			}
			var revision, count int
			var next time.Time
			if e = f.db.QueryRowContext(f.ctx, "SELECT revision,next_due_at FROM component_warmup_schedules WHERE schedule_id=?", saved.Schedule.ScheduleID).Scan(&revision, &next); e != nil {
				t.Fatal(e)
			}
			if e = f.db.QueryRowContext(f.ctx, "SELECT COUNT(*) FROM component_warmup_runs WHERE report_id=?", f.report.ID).Scan(&count); e != nil || count != 1 || revision != 2 || !next.After(now) {
				t.Fatalf("partial admission: runs=%d revision=%d next=%v err=%v", count, revision, next, e)
			}
		})
	}
}
func TestWarmupScheduleProcessWriter(t *testing.T) {
	driver := os.Getenv("SCHEDULE_PROOF_DRIVER")
	if driver == "" {
		t.Skip("subprocess entry point")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	db, e := sql.Open(driver, os.Getenv("SCHEDULE_PROOF_DSN"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	raw, e := os.ReadFile(os.Getenv("SCHEDULE_PROOF_INPUT"))
	if e != nil {
		t.Fatal(e)
	}
	var input scheduledAdmissionFixture
	if e = json.Unmarshal(raw, &input); e != nil {
		t.Fatal(e)
	}
	for {
		if _, e = os.Stat(os.Getenv("SCHEDULE_PROOF_GATE")); e == nil {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(time.Millisecond):
		}
	}
	ctx = sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner"})
	tr := &Transport{DB: db}
	if e = tr.consumeWarmupSchedule(ctx, &input.Schedule, &input.Run, input.Run.RunId, input.Now); e != nil {
		t.Fatal(e)
	}
}
