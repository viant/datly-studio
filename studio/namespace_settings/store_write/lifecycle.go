package store_write

import (
	context "context"
	"fmt"
	"github.com/viant/datly-studio/studio/settings"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
)

// NamespaceSettingsRules customizes role Input.Namespaces.
type NamespaceSettingsRules struct{}

func NamespaceSettingsRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*NamespaceSettingsRules)(nil)).Elem()
}

var (
	NamespaceSettingsRulesHooks = new(NamespaceSettingsRules)
	NamespaceSettingsRulesDatly = NamespaceSettingsRulesDatlyType()
)

func (hooks *NamespaceSettingsRules) Init(ctx context.Context, entity *StoredNamespace, state xhandler.LifecycleContext[StoredNamespace, xhandler.NoParent, Output]) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if entity == nil || state.Previous == nil || entity.Etag == nil || state.Previous.Etag == nil || *entity.Etag != *state.Previous.Etag {
		return &xhandler.Conflict{Entity: "namespace", Field: "etag", Reason: "expected namespace token does not match"}
	}
	if entity.UpdatedAt.IsZero() || len(entity.Entries) == 0 {
		return fmt.Errorf("settings update requires timestamp and at least one setting")
	}
	next := *entity.Etag + 1
	if next <= 0 {
		return fmt.Errorf("namespace token exhausted")
	}
	entity.SetEtag(&next)
	seen := map[string]bool{}
	for _, entry := range entity.Entries {
		if entry == nil || seen[entry.SettingKey] {
			return fmt.Errorf("duplicate or missing setting")
		}
		seen[entry.SettingKey] = true
	}
	return nil
}
func (hooks *NamespaceSettingsRules) Validate(ctx context.Context, entity *StoredNamespace, state xhandler.LifecycleContext[StoredNamespace, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *NamespaceSettingsRules) AfterSequence(ctx context.Context, entity *StoredNamespace, state xhandler.LifecycleContext[StoredNamespace, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *NamespaceSettingsRules) AfterQueue(ctx context.Context, entity *StoredNamespace, state xhandler.LifecycleContext[StoredNamespace, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *NamespaceSettingsRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}

// SettingRules customizes role Input.Namespaces.Entries.
type SettingRules struct{}

func SettingRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*SettingRules)(nil)).Elem()
}

var (
	SettingRulesHooks = new(SettingRules)
	SettingRulesDatly = SettingRulesDatlyType()
)

func (hooks *SettingRules) Init(ctx context.Context, entity *StoredSetting, state xhandler.LifecycleContext[StoredSetting, StoredNamespace, Output]) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if entity == nil || state.Parent == nil || entity.OwnerId != state.Parent.OwnerId || entity.Namespace != state.Parent.Name {
		return fmt.Errorf("setting must belong to its namespace")
	}
	if entity.UpdatedBy == "" || !entity.UpdatedAt.Equal(state.Parent.UpdatedAt) {
		return fmt.Errorf("setting audit fields are required")
	}
	_, err := settings.Decode(entity.SettingKey, entity.ValueJson)
	return err
}
func (hooks *SettingRules) Validate(ctx context.Context, entity *StoredSetting, state xhandler.LifecycleContext[StoredSetting, StoredNamespace, Output]) error {
	return nil
}
func (hooks *SettingRules) AfterSequence(ctx context.Context, entity *StoredSetting, state xhandler.LifecycleContext[StoredSetting, StoredNamespace, Output]) error {
	return nil
}
func (hooks *SettingRules) AfterQueue(ctx context.Context, entity *StoredSetting, state xhandler.LifecycleContext[StoredSetting, StoredNamespace, Output]) error {
	return nil
}
