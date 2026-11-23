package store_admit

import (
	context "context"
	"fmt"
	"github.com/viant/datly-studio/internal/warmupschedule"
	"github.com/viant/datly-studio/sdk/executionguard"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
)

// ScheduleRules customizes role Input.Schedules.
type ScheduleRules struct{}

func ScheduleRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*ScheduleRules)(nil)).Elem()
}

var (
	ScheduleRulesHooks = new(ScheduleRules)
	ScheduleRulesDatly = ScheduleRulesDatlyType()
)

func (hooks *ScheduleRules) Init(ctx context.Context, entity *StoredSchedule, state xhandler.LifecycleContext[StoredSchedule, xhandler.NoParent, Output]) error {
	p := state.Previous
	if entity == nil || p == nil || p.Revision == nil || entity.Revision == nil || *p.Revision != *entity.Revision || !p.Enabled || p.NextDueAt.After(entity.UpdatedAt) {
		return &xhandler.Conflict{Entity: "warmup_schedule", Field: "revision", Reason: "schedule changed or is not due"}
	}
	if p.NamespaceId != entity.NamespaceId || p.ReportId != entity.ReportId || p.VersionNo != entity.VersionNo || p.IntervalSeconds != entity.IntervalSeconds || !p.FirstRunAt.Equal(entity.FirstRunAt) || !entity.Enabled || p.CreatedBy != entity.CreatedBy || !p.CreatedAt.Equal(entity.CreatedAt) || entity.UpdatedBy == "" || entity.LastRunId == nil {
		return fmt.Errorf("schedule admission cannot change configuration or ownership")
	}
	next, e := warmupschedule.NextDue(p.NextDueAt, entity.UpdatedAt, p.IntervalSeconds)
	if e != nil {
		return e
	}
	if !next.Equal(entity.NextDueAt) {
		return fmt.Errorf("invalid next occurrence")
	}
	if len(entity.Run) > 1 {
		return fmt.Errorf("an occurrence accepts at most one run")
	}
	for _, r := range entity.Run {
		if r == nil || r.RunId != *entity.LastRunId || r.NamespaceId != p.NamespaceId || r.ReportId != p.ReportId || r.VersionNo != p.VersionNo || r.Status != "accepted" || r.RequestedBy != entity.UpdatedBy || r.DeadlineAt == nil || !r.DeadlineAt.After(r.RequestedAt) || !r.RequestedAt.Equal(entity.UpdatedAt) {
			return fmt.Errorf("invalid accepted schedule run")
		}
	}
	n := *entity.Revision + 1
	entity.SetRevision(&n)
	return nil
}
func (hooks *ScheduleRules) Validate(ctx context.Context, entity *StoredSchedule, state xhandler.LifecycleContext[StoredSchedule, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *ScheduleRules) AfterSequence(ctx context.Context, entity *StoredSchedule, state xhandler.LifecycleContext[StoredSchedule, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *ScheduleRules) AfterQueue(ctx context.Context, entity *StoredSchedule, state xhandler.LifecycleContext[StoredSchedule, xhandler.NoParent, Output]) error {
	return executionguard.Check(ctx)
}
func (hooks *ScheduleRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
