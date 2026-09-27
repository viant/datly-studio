package store_insert

import (
	context "context"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
)

// PredicateInsertRules customizes role Input.AuthorizationPredicates.
type PredicateInsertRules struct{}

func PredicateInsertRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*PredicateInsertRules)(nil)).Elem()
}

var (
	PredicateInsertRulesHooks = new(PredicateInsertRules)
	PredicateInsertRulesDatly = PredicateInsertRulesDatlyType()
)

func (hooks *PredicateInsertRules) Init(ctx context.Context, entity *StoredAuthorizationPredicate, state xhandler.LifecycleContext[StoredAuthorizationPredicate, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PredicateInsertRules) Validate(ctx context.Context, entity *StoredAuthorizationPredicate, state xhandler.LifecycleContext[StoredAuthorizationPredicate, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PredicateInsertRules) AfterSequence(ctx context.Context, entity *StoredAuthorizationPredicate, state xhandler.LifecycleContext[StoredAuthorizationPredicate, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PredicateInsertRules) AfterQueue(ctx context.Context, entity *StoredAuthorizationPredicate, state xhandler.LifecycleContext[StoredAuthorizationPredicate, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *PredicateInsertRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
