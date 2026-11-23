package store_admit

import (
	"context"
	existing "github.com/viant/datly-studio/studio/report_warmup_runs/store_write"
	xhandler "github.com/viant/xdatly/handler"
	"reflect"
)

// AcceptedRunRules delegates creation invariants to the existing warmup writer.
// The scheduled admission graph may insert a run, never replace an existing one.
type AcceptedRunRules struct{}

func AcceptedRunRulesDatlyType() reflect.Type { return reflect.TypeFor[AcceptedRunRules]() }

var AcceptedRunRulesDatly = AcceptedRunRulesDatlyType()

func (*AcceptedRunRules) Init(ctx context.Context, row *AcceptedRun, state xhandler.LifecycleContext[AcceptedRun, StoredSchedule, Output]) error {
	if state.Previous != nil {
		return &xhandler.Conflict{Entity: "warmup_run", Field: "run_id", Reason: "scheduled warmup run already exists"}
	}
	return (&existing.WarmupRunRules{}).Init(ctx, &existing.StoredWarmupRun{NamespaceId: row.NamespaceId, RunId: row.RunId, Status: row.Status, CreatedAt: row.CreatedAt, CreatedBy: row.CreatedBy, UpdatedAt: row.UpdatedAt, UpdatedBy: row.UpdatedBy, Has: &existing.StoredWarmupRunHas{NamespaceId: true}}, xhandler.LifecycleContext[existing.StoredWarmupRun, xhandler.NoParent, existing.Output]{})
}
func (*AcceptedRunRules) Validate(ctx context.Context, row *AcceptedRun, state xhandler.LifecycleContext[AcceptedRun, StoredSchedule, Output]) error {
	return nil
}
func (*AcceptedRunRules) AfterSequence(ctx context.Context, row *AcceptedRun, state xhandler.LifecycleContext[AcceptedRun, StoredSchedule, Output]) error {
	return nil
}
func (*AcceptedRunRules) AfterQueue(ctx context.Context, row *AcceptedRun, state xhandler.LifecycleContext[AcceptedRun, StoredSchedule, Output]) error {
	return nil
}
func (*AcceptedRunRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
