package store_write

import (
	context "context"
	"fmt"
	"github.com/viant/datly-studio/internal/namespaceaccess"
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
	if entity == nil || entity.ScheduleId == "" || entity.ReportId == "" || entity.VersionNo <= 0 || entity.Revision == nil || entity.UpdatedBy == "" || entity.UpdatedAt.IsZero() {
		return fmt.Errorf("complete schedule identity, expected revision and audit are required")
	}
	if e := namespaceaccess.ValidateResourceOwnership(entity.NamespaceId, ""); e != nil {
		return e
	}
	p := state.Previous
	if p == nil {
		if *entity.Revision != 0 || entity.ShouldDelete {
			return &xhandler.Conflict{Entity: "warmup_schedule", Field: "revision", Reason: "schedule no longer exists"}
		}
		if entity.CreatedBy == "" || entity.CreatedAt.IsZero() {
			return fmt.Errorf("schedule creation audit is required")
		}
	} else {
		if p.Revision == nil || *p.Revision != *entity.Revision {
			return &xhandler.Conflict{Entity: "warmup_schedule", Field: "revision", Reason: "schedule changed; reload before saving"}
		}
		if p.NamespaceId != entity.NamespaceId || p.ReportId != entity.ReportId || p.VersionNo != entity.VersionNo || p.CreatedBy != entity.CreatedBy || !p.CreatedAt.Equal(entity.CreatedAt) {
			return fmt.Errorf("schedule ownership and creation audit are immutable")
		}
	}
	if e := warmupschedule.Validate(entity.IntervalSeconds, entity.FirstRunAt); e != nil {
		return e
	}
	if entity.NextDueAt.IsZero() {
		return fmt.Errorf("next schedule occurrence is required")
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
