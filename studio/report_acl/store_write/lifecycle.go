package store_write

import (
	"context"
	"fmt"
	"github.com/viant/datly-studio/internal/namespaceaccess"
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
	if row == nil || row.Has == nil || !row.Has.NamespaceId {
		return fmt.Errorf("ACL namespace ownership is required")
	}
	previousNamespace := ""
	if state.Previous != nil {
		previousNamespace = state.Previous.NamespaceId
	}
	if err := namespaceaccess.ValidateResourceOwnership(row.NamespaceId, previousNamespace); err != nil {
		return err
	}
	if state.Previous == nil || row.ShouldDelete {
		return nil
	}
	if row.Etag == nil {
		return &xhandler.Conflict{Entity: "component_acl", Field: "etag", Reason: "expected token is missing"}
	}
	next := *row.Etag + 1
	row.SetEtag(&next)
	return nil
}

func (*ACLStoreRules) Validate(context.Context, *StoredACL, xhandler.LifecycleContext[StoredACL, xhandler.NoParent, Output]) error {
	return nil
}
