package store_insert

import (
	context "context"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
)

// PublicationEventRules customizes role Input.Events.
type PublicationEventRules struct{}

func PublicationEventRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*PublicationEventRules)(nil)).Elem()
}

var (
	PublicationEventRulesHooks = new(PublicationEventRules)
	PublicationEventRulesDatly = PublicationEventRulesDatlyType()
)

func (hooks *PublicationEventRules) Init(_ context.Context, entity *StoredEvent, _ xhandler.LifecycleContext[StoredEvent, xhandler.NoParent, Output]) error {
	if entity == nil || entity.EventId == "" || entity.ReportId == "" || entity.OwnerId == "" ||
		entity.RequestedBy == "" || entity.OccurredAt.IsZero() {
		return fmt.Errorf("publication event identity and audit fields are required")
	}
	switch entity.Operation {
	case "publish", "rollback", "unpublish":
	default:
		return fmt.Errorf("invalid publication event operation %q", entity.Operation)
	}
	if entity.Status != "succeeded" && entity.Status != "failed" {
		return fmt.Errorf("invalid publication event status %q", entity.Status)
	}
	return nil
}
func (hooks *PublicationEventRules) Validate(ctx context.Context, entity *StoredEvent, state xhandler.LifecycleContext[StoredEvent, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PublicationEventRules) AfterSequence(ctx context.Context, entity *StoredEvent, state xhandler.LifecycleContext[StoredEvent, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PublicationEventRules) AfterQueue(ctx context.Context, entity *StoredEvent, state xhandler.LifecycleContext[StoredEvent, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PublicationEventRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
