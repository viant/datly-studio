package create

import (
	"context"
	"github.com/viant/datly-studio/studio/namespaces/change"
	xhandler "github.com/viant/xdatly/handler"
	"reflect"
)

type NamespaceCreateRules struct{}

func NamespaceCreateRulesDatlyType() reflect.Type { return reflect.TypeFor[NamespaceCreateRules]() }

var NamespaceCreateRulesDatly = NamespaceCreateRulesDatlyType()

func (*NamespaceCreateRules) Init(context.Context, *NamespaceRecord, xhandler.LifecycleContext[NamespaceRecord, xhandler.NoParent, NamespaceCreateOutput]) error {
	return nil
}
func (*NamespaceCreateRules) Validate(context.Context, *NamespaceRecord, xhandler.LifecycleContext[NamespaceRecord, xhandler.NoParent, NamespaceCreateOutput]) error {
	return nil
}
func (*NamespaceCreateRules) AfterSequence(context.Context, *NamespaceRecord, xhandler.LifecycleContext[NamespaceRecord, xhandler.NoParent, NamespaceCreateOutput]) error {
	return nil
}
func (*NamespaceCreateRules) AfterQueue(context.Context, *NamespaceRecord, xhandler.LifecycleContext[NamespaceRecord, xhandler.NoParent, NamespaceCreateOutput]) error {
	return nil
}
func (*NamespaceCreateRules) Finalize(ctx context.Context, input *NamespaceCreateInput, output *NamespaceCreateOutput, outcome xhandler.Outcome) error {
	if !outcome.CommitConfirmed() || output == nil || output.Data == nil {
		return nil
	}
	row := output.Data
	err := change.Notify(ctx, []change.Identity{{NamespaceID: row.NamespaceId, OwnerID: row.OwnerId, Name: row.Name}})
	return err
}
