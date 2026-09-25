package store_activate

import (
	context "context"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
	"strings"
)

// PublicationActivationRules customizes role Input.Publications.
type PublicationActivationRules struct {
	Input *Input `bind:"kind=input,required"`
}

func PublicationActivationRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*PublicationActivationRules)(nil)).Elem()
}

var (
	PublicationActivationRulesHooks = new(PublicationActivationRules)
	PublicationActivationRulesDatly = PublicationActivationRulesDatlyType()
)

func (hooks *PublicationActivationRules) Init(_ context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	if hooks.Input == nil || entity == nil || strings.TrimSpace(entity.ReportId) == "" ||
		entity.DesiredGeneration == nil || entity.ActivatedAt == nil || entity.ActivatedAt.IsZero() ||
		entity.Has == nil || !entity.Has.ReportId || !entity.Has.DesiredGeneration ||
		!entity.Has.ActivatedAt || hooks.Input.Generation <= 0 || hooks.Input.VersionNo <= 0 {
		return fmt.Errorf("publication activation requires exact staged identity, version, generation and timestamp")
	}
	previous := state.Previous
	if previous == nil || previous.DesiredGeneration == nil || *previous.DesiredGeneration != *entity.DesiredGeneration ||
		*entity.DesiredGeneration != hooks.Input.Generation || previous.DesiredVersionNo == nil ||
		*previous.DesiredVersionNo != hooks.Input.VersionNo || previous.PublicationStatus != "pending" {
		return &xhandler.Conflict{Entity: "report_publication", Field: "desired_generation", Reason: "staged publication changed before activation"}
	}
	version := hooks.Input.VersionNo
	generation := hooks.Input.Generation
	entity.SetActiveVersionNo(&version)
	entity.SetActiveGeneration(&generation)
	entity.SetPublicationStatus("active")
	entity.SetActivatedAt(entity.ActivatedAt)
	entity.SetFailureJson(nil)
	return nil
}
func (hooks *PublicationActivationRules) Validate(ctx context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PublicationActivationRules) AfterSequence(ctx context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PublicationActivationRules) AfterQueue(ctx context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PublicationActivationRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
