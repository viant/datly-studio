package store_insert

import (
	context "context"
	"fmt"
	"github.com/viant/datly-studio/studio/namespaces/change"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
)

// NamespaceInsertRules customizes role Input.Namespaces.
type NamespaceInsertRules struct{}

func NamespaceInsertRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*NamespaceInsertRules)(nil)).Elem()
}

var (
	NamespaceInsertRulesHooks = new(NamespaceInsertRules)
	NamespaceInsertRulesDatly = NamespaceInsertRulesDatlyType()
)

func (hooks *NamespaceInsertRules) Init(_ context.Context, entity *StoredNamespace, _ xhandler.LifecycleContext[StoredNamespace, xhandler.NoParent, Output]) error {
	if entity == nil || entity.OwnerId == "" || entity.Name == "" || entity.Title == "" ||
		entity.Status != "active" || entity.Etag == nil || *entity.Etag != 1 ||
		entity.CreatedAt == nil || entity.UpdatedAt == nil || entity.DeletedAt != nil {
		return fmt.Errorf("new namespace requires owner, name, title, active state, timestamps and etag 1")
	}
	return nil
}
func (hooks *NamespaceInsertRules) Validate(ctx context.Context, entity *StoredNamespace, state xhandler.LifecycleContext[StoredNamespace, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *NamespaceInsertRules) AfterSequence(ctx context.Context, entity *StoredNamespace, state xhandler.LifecycleContext[StoredNamespace, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *NamespaceInsertRules) AfterQueue(ctx context.Context, entity *StoredNamespace, state xhandler.LifecycleContext[StoredNamespace, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *NamespaceInsertRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	if !outcome.CommitConfirmed() || output == nil {
		return nil
	}
	identities := make([]change.Identity, 0, len(output.Data))
	for _, row := range output.Data {
		if row != nil {
			identities = append(identities, change.Identity{NamespaceID: row.NamespaceId, OwnerID: row.OwnerId, Name: row.Name})
		}
	}
	return change.Notify(ctx, identities)
}
