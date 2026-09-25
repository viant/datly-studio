package store_status

import (
	context "context"
	"errors"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
)

// ConnectorStatusRules customizes role Input.Connectors.
type ConnectorStatusRules struct {
	Input *Input `bind:"kind=input,required"`
}

var ErrProbeRequired = errors.New("connector must pass a connectivity test before activation")

func ConnectorStatusRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*ConnectorStatusRules)(nil)).Elem()
}

var (
	ConnectorStatusRulesHooks = new(ConnectorStatusRules)
	ConnectorStatusRulesDatly = ConnectorStatusRulesDatlyType()
)

func (hooks *ConnectorStatusRules) Init(_ context.Context, entity *StoredConnector, state xhandler.LifecycleContext[StoredConnector, xhandler.NoParent, Output]) error {
	if hooks.Input == nil || entity == nil || entity.Name == "" || entity.UpdatedAt == nil || entity.Etag == nil {
		return fmt.Errorf("connector status requires operation, name, timestamp and expected etag")
	}
	previous := state.Previous
	if previous == nil || previous.DeletedAt != nil {
		return &xhandler.Conflict{Entity: "connector", Field: "name", Reason: "connector is absent or deleted"}
	}
	if previous.Etag == nil || *previous.Etag != *entity.Etag {
		return &xhandler.Conflict{Entity: "connector", Field: "etag", Reason: "expected etag does not match"}
	}
	switch hooks.Input.Operation {
	case "activate":
		if previous.LastTestStatus == nil || *previous.LastTestStatus != "passed" {
			return ErrProbeRequired
		}
		entity.SetStatus("active")
	case "disable":
		entity.SetStatus("disabled")
	default:
		return fmt.Errorf("unsupported connector status operation %q", hooks.Input.Operation)
	}
	next := *entity.Etag + 1
	entity.SetEtag(&next)
	return nil
}
func (hooks *ConnectorStatusRules) Validate(ctx context.Context, entity *StoredConnector, state xhandler.LifecycleContext[StoredConnector, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *ConnectorStatusRules) AfterSequence(ctx context.Context, entity *StoredConnector, state xhandler.LifecycleContext[StoredConnector, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *ConnectorStatusRules) AfterQueue(ctx context.Context, entity *StoredConnector, state xhandler.LifecycleContext[StoredConnector, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *ConnectorStatusRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
