package store_insert

import (
	context "context"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
)

// ConnectorInsertRules customizes role Input.Connectors.
type ConnectorInsertRules struct{}

func ConnectorInsertRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*ConnectorInsertRules)(nil)).Elem()
}

var (
	ConnectorInsertRulesHooks = new(ConnectorInsertRules)
	ConnectorInsertRulesDatly = ConnectorInsertRulesDatlyType()
)

func (hooks *ConnectorInsertRules) Init(_ context.Context, entity *StoredConnector, _ xhandler.LifecycleContext[StoredConnector, xhandler.NoParent, Output]) error {
	if entity == nil || entity.Name == "" || entity.Driver == "" || entity.OwnerId == "" ||
		entity.Status != "draft" || entity.Etag != 1 || entity.CreatedAt.IsZero() ||
		entity.UpdatedAt.IsZero() {
		return fmt.Errorf("new connector requires name, driver, owner, draft state, timestamps and etag 1")
	}
	return nil
}
func (hooks *ConnectorInsertRules) Validate(ctx context.Context, entity *StoredConnector, state xhandler.LifecycleContext[StoredConnector, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *ConnectorInsertRules) AfterSequence(ctx context.Context, entity *StoredConnector, state xhandler.LifecycleContext[StoredConnector, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *ConnectorInsertRules) AfterQueue(ctx context.Context, entity *StoredConnector, state xhandler.LifecycleContext[StoredConnector, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *ConnectorInsertRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
