package store_write

import (
	"context"
	"reflect"

	xhandler "github.com/viant/xdatly/handler"
)

// PredicateStoreRules advances the persisted etag after the writer checks the
// caller's expected value against the previous row.
type PredicateStoreRules struct{}

var PredicateStoreRulesHooks = new(PredicateStoreRules)

func PredicateStoreRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*PredicateStoreRules)(nil)).Elem()
}

var PredicateStoreRulesDatlyLinkedType = PredicateStoreRulesDatlyType()

func (*PredicateStoreRules) Init(_ context.Context, row *StoredAuthorizationPredicate, state xhandler.LifecycleContext[StoredAuthorizationPredicate, xhandler.NoParent, Output]) error {
	if state.Previous == nil {
		return nil
	}
	if row.Etag == nil {
		return &xhandler.Conflict{Entity: "authorization_predicate", Field: "etag", Reason: "expected token is missing"}
	}
	next := *row.Etag + 1
	row.Etag = &next
	if row.Has == nil {
		row.Has = &StoredAuthorizationPredicateHas{}
	}
	row.Has.Etag = true
	return nil
}

func (*PredicateStoreRules) Validate(context.Context, *StoredAuthorizationPredicate, xhandler.LifecycleContext[StoredAuthorizationPredicate, xhandler.NoParent, Output]) error {
	return nil
}
