package store_config

import (
	"bytes"
	context "context"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
)

// ConnectorConfigRules customizes role Input.Connectors.
type ConnectorConfigRules struct{}

func ConnectorConfigRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*ConnectorConfigRules)(nil)).Elem()
}

var (
	ConnectorConfigRulesHooks = new(ConnectorConfigRules)
	ConnectorConfigRulesDatly = ConnectorConfigRulesDatlyType()
)

func (hooks *ConnectorConfigRules) Init(_ context.Context, entity *StoredConnector, state xhandler.LifecycleContext[StoredConnector, xhandler.NoParent, Output]) error {
	if entity == nil || entity.Name == "" || entity.Etag == nil || entity.UpdatedAt == nil ||
		entity.Has == nil || !entity.Has.Name || !entity.Has.Driver || !entity.Has.DsnTemplate ||
		!entity.Has.SecretRef || !entity.Has.Description || !entity.Has.OptionsJson ||
		!entity.Has.Etag || !entity.Has.UpdatedAt {
		return fmt.Errorf("connector config requires a complete configuration snapshot and expected etag")
	}
	previous := state.Previous
	if previous == nil || previous.DeletedAt != nil {
		return &xhandler.Conflict{Entity: "connector", Field: "name", Reason: "connector is absent or deleted"}
	}
	if previous.Etag == nil || *previous.Etag != *entity.Etag {
		return &xhandler.Conflict{Entity: "connector", Field: "etag", Reason: "expected etag does not match"}
	}
	if entity.Driver != previous.Driver || stringValue(entity.DsnTemplate) != stringValue(previous.DsnTemplate) ||
		stringValue(entity.SecretRef) != stringValue(previous.SecretRef) || !bytes.Equal(entity.OptionsJson, previous.OptionsJson) {
		entity.SetStatus("draft")
		entity.SetLastTestStatus(nil)
		entity.SetLastTestErrorCode(nil)
		entity.SetLastTestedAt(nil)
	}
	next := *entity.Etag + 1
	entity.SetEtag(&next)
	return nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func (hooks *ConnectorConfigRules) Validate(ctx context.Context, entity *StoredConnector, state xhandler.LifecycleContext[StoredConnector, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *ConnectorConfigRules) AfterSequence(ctx context.Context, entity *StoredConnector, state xhandler.LifecycleContext[StoredConnector, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *ConnectorConfigRules) AfterQueue(ctx context.Context, entity *StoredConnector, state xhandler.LifecycleContext[StoredConnector, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *ConnectorConfigRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
