package store_delete

import (
	context "context"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
	"strings"
)

// PublicationDeleteRules customizes role Input.Publications.
type PublicationDeleteRules struct{}

func PublicationDeleteRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*PublicationDeleteRules)(nil)).Elem()
}

var (
	PublicationDeleteRulesHooks = new(PublicationDeleteRules)
	PublicationDeleteRulesDatly = PublicationDeleteRulesDatlyType()
)

func (hooks *PublicationDeleteRules) Init(_ context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	if entity == nil || strings.TrimSpace(entity.ReportId) == "" || entity.DesiredGeneration == nil ||
		!entity.ShouldDelete || entity.Has == nil || !entity.Has.ReportId ||
		!entity.Has.DesiredGeneration || !entity.Has.ShouldDelete {
		return fmt.Errorf("publication delete requires exact report, generation, and delete marker")
	}
	previous := state.Previous
	if previous == nil || previous.ReportId != entity.ReportId || previous.DesiredGeneration == nil ||
		*previous.DesiredGeneration != *entity.DesiredGeneration || previous.PublicationStatus != "unpublishing" {
		return &xhandler.Conflict{Entity: "report_publication", Field: "desired_generation", Reason: "staged unpublish changed before deletion"}
	}
	return nil
}
func (hooks *PublicationDeleteRules) Validate(ctx context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PublicationDeleteRules) AfterSequence(ctx context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PublicationDeleteRules) AfterQueue(ctx context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PublicationDeleteRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
