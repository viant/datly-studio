package store_stage

import (
	context "context"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
	"strings"
)

// PublicationStageRules customizes role Input.Publications.
type PublicationStageRules struct {
	Input *Input `bind:"kind=input,required"`
}

func PublicationStageRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*PublicationStageRules)(nil)).Elem()
}

var (
	PublicationStageRulesHooks = new(PublicationStageRules)
	PublicationStageRulesDatly = PublicationStageRulesDatlyType()
)

func (hooks *PublicationStageRules) Init(_ context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	if hooks.Input == nil || entity == nil || strings.TrimSpace(entity.ReportId) == "" ||
		entity.DesiredGeneration == nil || entity.DesiredVersionNo == nil || *entity.DesiredVersionNo <= 0 ||
		entity.PublishedAt == nil || entity.PublishedAt.IsZero() ||
		entity.Has == nil || !entity.Has.ReportId || !entity.Has.DesiredVersionNo ||
		!entity.Has.DesiredGeneration || !entity.Has.PublicationStatus ||
		!entity.Has.RuntimeRevision || !entity.Has.SpecHash || !entity.Has.PublishedBy ||
		!entity.Has.PublishedAt || !entity.Has.FailureJson {
		return fmt.Errorf("publication restage requires exact identity, expected generation and complete transition")
	}
	previous := state.Previous
	if previous == nil || previous.DesiredGeneration == nil || *previous.DesiredGeneration != *entity.DesiredGeneration ||
		previous.PublicationStatus == "pending" || previous.PublicationStatus == "unpublishing" {
		return &xhandler.Conflict{Entity: "report_publication", Field: "desired_generation", Reason: "publication changed or transition already staged"}
	}
	if hooks.Input.NextGeneration <= *entity.DesiredGeneration || entity.PublicationStatus != "pending" ||
		entity.RuntimeRevision == nil || strings.TrimSpace(*entity.RuntimeRevision) == "" ||
		strings.TrimSpace(entity.PublishedBy) == "" || entity.FailureJson != nil || strings.TrimSpace(entity.SpecHash) == "" {
		return fmt.Errorf("publication restage requires a newer pending generation without failure evidence")
	}
	next := hooks.Input.NextGeneration
	entity.SetDesiredGeneration(&next)
	return nil
}
func (hooks *PublicationStageRules) Validate(ctx context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PublicationStageRules) AfterSequence(ctx context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PublicationStageRules) AfterQueue(ctx context.Context, entity *StoredPublication, state xhandler.LifecycleContext[StoredPublication, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PublicationStageRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
