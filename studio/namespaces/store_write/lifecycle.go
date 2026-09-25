package store_write

import (
	context "context"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
)

// NamespaceStoreRules customizes role Input.Namespaces.
type NamespaceStoreRules struct{}

func NamespaceStoreRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*NamespaceStoreRules)(nil)).Elem()
}

var (
	NamespaceStoreRulesHooks = new(NamespaceStoreRules)
	NamespaceStoreRulesDatly = NamespaceStoreRulesDatlyType()
)

func (hooks *NamespaceStoreRules) Init(_ context.Context, entity *StoredNamespace, state xhandler.LifecycleContext[StoredNamespace, xhandler.NoParent, Output]) error {
	if entity == nil || entity.OwnerId == "" || entity.Name == "" {
		return fmt.Errorf("namespace owner and name are required")
	}
	if state.Previous == nil {
		if entity.Etag == nil || *entity.Etag != 1 || entity.Status != "active" ||
			entity.Title == "" || entity.CreatedAt == nil || entity.UpdatedAt == nil || entity.DeletedAt != nil {
			return fmt.Errorf("new namespace requires active state, title, timestamps and etag 1")
		}
		return nil
	}
	if entity.Etag == nil {
		return &xhandler.Conflict{Entity: "namespace", Field: "etag", Reason: "expected token is missing"}
	}
	if entity.UpdatedAt == nil {
		return fmt.Errorf("namespace updated_at is required")
	}
	if entity.DeletedAt != nil && entity.Status != "archived" {
		return fmt.Errorf("deleted namespace must be archived")
	}
	next := *entity.Etag + 1
	entity.Etag = &next
	if entity.Has == nil {
		entity.Has = &StoredNamespaceHas{}
	}
	entity.Has.Etag = true
	return nil
}
func (hooks *NamespaceStoreRules) Validate(ctx context.Context, entity *StoredNamespace, state xhandler.LifecycleContext[StoredNamespace, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *NamespaceStoreRules) AfterSequence(ctx context.Context, entity *StoredNamespace, state xhandler.LifecycleContext[StoredNamespace, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *NamespaceStoreRules) AfterQueue(ctx context.Context, entity *StoredNamespace, state xhandler.LifecycleContext[StoredNamespace, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *NamespaceStoreRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
