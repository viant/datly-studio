package store_state

import (
	context "context"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
	"strings"
)

// VersionStateRules customizes role Input.Versions.
type VersionStateRules struct {
	Input *Input `bind:"kind=input,required"`
}

func VersionStateRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*VersionStateRules)(nil)).Elem()
}

var (
	VersionStateRulesHooks = new(VersionStateRules)
	VersionStateRulesDatly = VersionStateRulesDatlyType()
)

func (hooks *VersionStateRules) Init(_ context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	if hooks.Input == nil || hooks.Input.TargetVersionNo <= 0 || entity == nil ||
		strings.TrimSpace(entity.ReportId) == "" || entity.VersionNo <= 0 ||
		entity.Has == nil || !entity.Has.ReportId || !entity.Has.VersionNo || !entity.Has.State {
		return fmt.Errorf("version state requires exact report, version and expected state")
	}
	previous := state.Previous
	if previous == nil || previous.State != entity.State {
		return &xhandler.Conflict{Entity: "report_version", Field: "state", Reason: "version state changed"}
	}
	switch hooks.Input.Operation {
	case "publish":
		if entity.VersionNo != hooks.Input.TargetVersionNo || entity.PublishedAt == nil ||
			entity.PublishedAt.IsZero() || !entity.Has.PublishedAt {
			return fmt.Errorf("publication requires exact target version and timestamp")
		}
		entity.SetState("published")
		entity.SetPublishedAt(entity.PublishedAt)
	case "supersede":
		if entity.VersionNo == hooks.Input.TargetVersionNo || previous.State != "published" {
			return fmt.Errorf("supersede requires another published version")
		}
		entity.SetState("superseded")
	case "unpublish":
		if previous.State != "published" {
			return fmt.Errorf("unpublish requires a published version")
		}
		entity.SetState("superseded")
	default:
		return fmt.Errorf("unsupported version state operation %q", hooks.Input.Operation)
	}
	return nil
}
func (hooks *VersionStateRules) Validate(ctx context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *VersionStateRules) AfterSequence(ctx context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *VersionStateRules) AfterQueue(ctx context.Context, entity *StoredVersion, state xhandler.LifecycleContext[StoredVersion, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *VersionStateRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
