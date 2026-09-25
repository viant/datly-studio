package store_write

import (
	"context"
	"reflect"

	xhandler "github.com/viant/xdatly/handler"
)

// ACLStoreRules advances the token only for updates. The SDK owns subject and
// actor authorization; Datly owns the atomic persisted-token comparison.
type ACLStoreRules struct{}

var ACLStoreRulesHooks = new(ACLStoreRules)

func ACLStoreRulesDatlyType() reflect.Type { return reflect.TypeOf((*ACLStoreRules)(nil)).Elem() }

var ACLStoreRulesDatlyLinkedType = ACLStoreRulesDatlyType()

func (*ACLStoreRules) Init(_ context.Context, row *StoredACL, state xhandler.LifecycleContext[StoredACL, xhandler.NoParent, Output]) error {
	if state.Previous == nil || row.ShouldDelete {
		return nil
	}
	if row.Etag == nil {
		return &xhandler.Conflict{Entity: "report_acl", Field: "etag", Reason: "expected token is missing"}
	}
	next := *row.Etag + 1
	row.SetEtag(&next)
	return nil
}

func (*ACLStoreRules) Validate(context.Context, *StoredACL, xhandler.LifecycleContext[StoredACL, xhandler.NoParent, Output]) error {
	return nil
}
