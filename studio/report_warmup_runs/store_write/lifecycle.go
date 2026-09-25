package store_write

import (
	context "context"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
	"time"
)

// WarmupRunRules customizes role Input.Runs.
type WarmupRunRules struct{}

func WarmupRunRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*WarmupRunRules)(nil)).Elem()
}

var (
	WarmupRunRulesHooks = new(WarmupRunRules)
	WarmupRunRulesDatly = WarmupRunRulesDatlyType()
)

func (hooks *WarmupRunRules) Init(_ context.Context, entity *StoredWarmupRun, state xhandler.LifecycleContext[StoredWarmupRun, xhandler.NoParent, Output]) error {
	if entity == nil || entity.RunId == "" {
		return fmt.Errorf("warmup run id is required")
	}
	if entity.UpdatedAt == nil {
		return fmt.Errorf("warmup updated_at token is required")
	}
	if state.Previous == nil {
		if entity.CreatedAt == nil || entity.CreatedBy == nil || *entity.CreatedBy == "" ||
			entity.UpdatedBy == nil || *entity.UpdatedBy == "" {
			return fmt.Errorf("warmup creation audit fields are required")
		}
		if entity.Status != "accepted" {
			return fmt.Errorf("new warmup runs must start accepted")
		}
		return nil
	}
	if entity.UpdatedBy == nil || *entity.UpdatedBy == "" {
		return fmt.Errorf("warmup updated_by is required")
	}
	switch state.Previous.Status {
	case "accepted":
		if entity.Status != "running" && entity.Status != "failed" {
			return fmt.Errorf("unsupported warmup transition accepted -> %s", entity.Status)
		}
	case "running":
		if entity.Status != "completed" && entity.Status != "partial" && entity.Status != "failed" {
			return fmt.Errorf("unsupported warmup transition running -> %s", entity.Status)
		}
	default:
		return fmt.Errorf("warmup status %s is terminal", state.Previous.Status)
	}
	next := time.Now().UTC().Truncate(time.Microsecond)
	if !next.After(*entity.UpdatedAt) {
		next = entity.UpdatedAt.Add(time.Microsecond)
	}
	entity.UpdatedAt = &next
	if entity.Has == nil {
		entity.Has = &StoredWarmupRunHas{}
	}
	entity.Has.Status, entity.Has.UpdatedAt = true, true
	return nil
}
func (hooks *WarmupRunRules) Validate(ctx context.Context, entity *StoredWarmupRun, state xhandler.LifecycleContext[StoredWarmupRun, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *WarmupRunRules) AfterSequence(ctx context.Context, entity *StoredWarmupRun, state xhandler.LifecycleContext[StoredWarmupRun, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *WarmupRunRules) AfterQueue(ctx context.Context, entity *StoredWarmupRun, state xhandler.LifecycleContext[StoredWarmupRun, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *WarmupRunRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
