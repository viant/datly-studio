package store_repoint

import (
	context "context"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
	"strings"
)

// PublicationRepointRules customizes role Input.Publications.
type PublicationRepointRules struct {
	Input *Input `bind:"kind=input,required"`
}

func PublicationRepointRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*PublicationRepointRules)(nil)).Elem()
}

var (
	PublicationRepointRulesHooks = new(PublicationRepointRules)
	PublicationRepointRulesDatly = PublicationRepointRulesDatlyType()
)

func (hooks *PublicationRepointRules) Init(_ context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	if hooks.Input == nil || entity == nil || strings.TrimSpace(entity.ReportId) == "" ||
		strings.TrimSpace(hooks.Input.ExcludeReportId) == "" || entity.ReportId == hooks.Input.ExcludeReportId ||
		entity.ActiveGeneration == nil || hooks.Input.Generation <= 0 ||
		entity.Has == nil || !entity.Has.ReportId || !entity.Has.ActiveGeneration {
		return fmt.Errorf("active publication repoint requires another report, expected active generation and next generation")
	}
	previous := state.Previous
	if previous == nil || previous.PublicationStatus != "active" || previous.ActiveGeneration == nil ||
		*previous.ActiveGeneration != *entity.ActiveGeneration {
		return &xhandler.Conflict{Entity: "report_publication", Field: "active_generation", Reason: "other active publication changed"}
	}
	next := hooks.Input.Generation
	entity.SetActiveGeneration(&next)
	return nil
}
func (hooks *PublicationRepointRules) Validate(ctx context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PublicationRepointRules) AfterSequence(ctx context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PublicationRepointRules) AfterQueue(ctx context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PublicationRepointRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
