package stage_test

import (
	"context"
	"database/sql"
	complete "github.com/viant/datly-studio/studio/artifact_release/complete"
	fail "github.com/viant/datly-studio/studio/artifact_release/fail"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
	"reflect"
	"testing"
	"time"
)

func TestNativeCompleteAtomic(t *testing.T) {
	for _, lateFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit", true: "rollback"}[lateFailure], func(t *testing.T) {
			ctx := context.Background()
			db, server, stage := newStageCase(ctx, t, "")
			if err := invokeStage(ctx, server, stage); err != nil {
				t.Fatal(err)
			}
			if lateFailure {
				if _, err := db.ExecContext(ctx, `CREATE TRIGGER reject_completion BEFORE UPDATE ON artifact_release_operations BEGIN SELECT RAISE(ABORT,'synthetic completion failure');END;`); err != nil {
					t.Fatal(err)
				}
			}
			row := &complete.CompleteChannel{}
			row.SetScopeId(&stage.ScopeId)
			row.SetName(&stage.Name)
			row.SetRevision(0)
			now := time.Now().UTC()
			row.SetUpdatedAt(&now)
			row.SetUpdatedBy(&stage.Actor)
			op := &complete.CompleteOperation{}
			id := *stage.Stage[0].PendingOperationId
			op.SetOperationId(&id)
			op.SetScopeId(&stage.ScopeId)
			op.SetChannelName(&stage.Name)
			op.SetUpdatedAt(&now)
			op.SetCompletedAt(&now)
			op.SetUpdatedBy(&stage.Actor)
			status := "applied"
			op.SetStatus(&status)
			row.SetOperation([]*complete.CompleteOperation{op})
			input := &complete.Input{}
			input.SetScopeId(stage.ScopeId)
			input.SetName(stage.Name)
			input.SetOperationId(id)
			input.SetExpectedRevision(0)
			input.SetPendingStatus("pending")
			input.SetActor(stage.Actor)
			input.SetComplete([]*complete.CompleteChannel{row})
			pkg := reflect.TypeFor[complete.CompleteComponent]().PkgPath()
			_, err := server.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: pkg, Name: "complete"}, Route: spec.RouteRef{Method: "PATCH", Path: "/_studio/artifact-release-store/complete"}}, Input: input})
			if lateFailure && err == nil {
				t.Fatal("late failure accepted")
			}
			if !lateFailure && err != nil {
				t.Fatal(err)
			}
			var revision int64
			var pending, hash sql.NullString
			var actual string
			if err := db.QueryRowContext(ctx, `SELECT revision,pending_operation_id,active_bundle_hash FROM artifact_release_channels`).Scan(&revision, &pending, &hash); err != nil {
				t.Fatal(err)
			}
			if err := db.QueryRowContext(ctx, `SELECT status FROM artifact_release_operations`).Scan(&actual); err != nil {
				t.Fatal(err)
			}
			expectedRevision := int64(0)
			if !lateFailure {
				expectedRevision = 1
				if !hash.Valid || hash.String != stage.TargetBundleHash {
					t.Fatal("active hash not promoted")
				}
			}
			if lateFailure {
				if !pending.Valid || actual != "pending" {
					t.Fatalf("partial completion pending=%v status=%s", pending, actual)
				}
			} else if pending.Valid || actual != status {
				t.Fatalf("incomplete outcome pending=%v status=%s", pending, actual)
			}
			if revision != expectedRevision {
				t.Fatalf("revision=%d expected=%d", revision, expectedRevision)
			}
		})
	}
}
func TestNativeFailAtomic(t *testing.T) {
	for _, lateFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit", true: "rollback"}[lateFailure], func(t *testing.T) {
			ctx := context.Background()
			db, server, stage := newStageCase(ctx, t, "")
			if err := invokeStage(ctx, server, stage); err != nil {
				t.Fatal(err)
			}
			if lateFailure {
				if _, err := db.ExecContext(ctx, `CREATE TRIGGER reject_completion BEFORE UPDATE ON artifact_release_operations BEGIN SELECT RAISE(ABORT,'synthetic completion failure');END;`); err != nil {
					t.Fatal(err)
				}
			}
			row := &fail.FailChannel{}
			row.SetScopeId(&stage.ScopeId)
			row.SetName(&stage.Name)
			row.SetRevision(0)
			now := time.Now().UTC()
			row.SetUpdatedAt(&now)
			row.SetUpdatedBy(&stage.Actor)
			op := &fail.FailOperation{}
			id := *stage.Stage[0].PendingOperationId
			op.SetOperationId(&id)
			op.SetScopeId(&stage.ScopeId)
			op.SetChannelName(&stage.Name)
			op.SetUpdatedAt(&now)
			op.SetCompletedAt(&now)
			op.SetUpdatedBy(&stage.Actor)
			status, code := "failed", "projection_failed"
			op.SetStatus(&status)
			op.SetFailureCode(&code)
			row.SetOperation([]*fail.FailOperation{op})
			input := &fail.Input{}
			input.SetScopeId(stage.ScopeId)
			input.SetName(stage.Name)
			input.SetOperationId(id)
			input.SetExpectedRevision(0)
			input.SetPendingStatus("pending")
			input.SetActor(stage.Actor)
			input.SetFail([]*fail.FailChannel{row})
			pkg := reflect.TypeFor[fail.FailComponent]().PkgPath()
			_, err := server.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: pkg, Name: "fail"}, Route: spec.RouteRef{Method: "PATCH", Path: "/_studio/artifact-release-store/fail"}}, Input: input})
			if lateFailure && err == nil {
				t.Fatal("late failure accepted")
			}
			if !lateFailure && err != nil {
				t.Fatal(err)
			}
			var revision int64
			var pending, hash sql.NullString
			var actual string
			if err := db.QueryRowContext(ctx, `SELECT revision,pending_operation_id,active_bundle_hash FROM artifact_release_channels`).Scan(&revision, &pending, &hash); err != nil {
				t.Fatal(err)
			}
			if err := db.QueryRowContext(ctx, `SELECT status FROM artifact_release_operations`).Scan(&actual); err != nil {
				t.Fatal(err)
			}
			expectedRevision := int64(0)
			if lateFailure {
				if !pending.Valid || actual != "pending" {
					t.Fatalf("partial completion pending=%v status=%s", pending, actual)
				}
			} else if pending.Valid || actual != status {
				t.Fatalf("incomplete outcome pending=%v status=%s", pending, actual)
			}
			if revision != expectedRevision {
				t.Fatalf("revision=%d expected=%d", revision, expectedRevision)
			}
		})
	}
}
