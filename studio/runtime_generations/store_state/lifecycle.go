package store_state

import (
	context "context"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
)

// GenerationStateRules customizes role Input.Generations.
type GenerationStateRules struct {
	Input *Input `bind:"kind=input,required"`
}

func GenerationStateRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*GenerationStateRules)(nil)).Elem()
}

var (
	GenerationStateRulesHooks = new(GenerationStateRules)
	GenerationStateRulesDatly = GenerationStateRulesDatlyType()
)

func (hooks *GenerationStateRules) Init(_ context.Context, entity *StoredGeneration, state xhandler.LifecycleContext[StoredGeneration, xhandler.NoParent, Output]) error {
	if hooks.Input == nil || hooks.Input.TargetGeneration <= 0 || entity == nil || entity.GenerationNo <= 0 ||
		entity.Has == nil || !entity.Has.GenerationNo || !entity.Has.Status {
		return fmt.Errorf("generation state requires exact identity and expected status")
	}
	previous := state.Previous
	if previous == nil || previous.Status != entity.Status {
		return &xhandler.Conflict{Entity: "runtime_generation", Field: "status", Reason: "generation status changed"}
	}
	switch hooks.Input.Operation {
	case "activate":
		if entity.GenerationNo != hooks.Input.TargetGeneration || previous.Status != "building" ||
			entity.ReportCount == nil || *entity.ReportCount < 0 || entity.ActivatedAt == nil ||
			entity.ActivatedAt.IsZero() || !entity.Has.ReportCount || !entity.Has.ActivatedAt {
			return fmt.Errorf("activation requires building target, report count and timestamp")
		}
		entity.SetStatus("active")
		entity.SetReportCount(entity.ReportCount)
		entity.SetActivatedAt(entity.ActivatedAt)
	case "retire":
		if entity.GenerationNo == hooks.Input.TargetGeneration || previous.Status != "active" ||
			entity.RetiredAt == nil || entity.RetiredAt.IsZero() || !entity.Has.RetiredAt {
			return fmt.Errorf("retirement requires another active generation and timestamp")
		}
		entity.SetStatus("retired")
		entity.SetRetiredAt(entity.RetiredAt)
	default:
		return fmt.Errorf("unsupported generation state operation %q", hooks.Input.Operation)
	}
	return nil
}
func (hooks *GenerationStateRules) Validate(ctx context.Context, entity *StoredGeneration, state xhandler.LifecycleContext[StoredGeneration, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *GenerationStateRules) AfterSequence(ctx context.Context, entity *StoredGeneration, state xhandler.LifecycleContext[StoredGeneration, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *GenerationStateRules) AfterQueue(ctx context.Context, entity *StoredGeneration, state xhandler.LifecycleContext[StoredGeneration, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *GenerationStateRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
