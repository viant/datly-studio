package control_write

import (
	context "context"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
)

// ScopeRules customizes role Input.Scopes.
type ScopeRules struct{}

func ScopeRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*ScopeRules)(nil)).Elem()
}

var (
	ScopeRulesHooks = new(ScopeRules)
	ScopeRulesDatly = ScopeRulesDatlyType()
)

func (hooks *ScopeRules) Init(ctx context.Context, entity *AuditScope, state xhandler.LifecycleContext[AuditScope, xhandler.NoParent, Output]) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if entity == nil || state.Previous == nil {
		return fmt.Errorf("active audit namespace is required")
	}
	if entity.NamespaceId == nil || state.Previous.NamespaceId == nil || *entity.NamespaceId != *state.Previous.NamespaceId {
		return fmt.Errorf("audit namespace identity changed")
	}
	if entity.Has != nil && entity.Has.AuditEnabled {
		if entity.UpdatedAt.IsZero() || entity.Etag != state.Previous.Etag {
			return fmt.Errorf("audit namespace update requires current revision and timestamp")
		}
		entity.SetEtag(entity.Etag + 1)
	}
	return nil
}

// ComponentRules customizes role Input.Scopes.Components.
type ComponentRules struct{}

func ComponentRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*ComponentRules)(nil)).Elem()
}

var (
	ComponentRulesHooks = new(ComponentRules)
	ComponentRulesDatly = ComponentRulesDatlyType()
)

func (hooks *ComponentRules) Init(ctx context.Context, entity *AuditComponent, state xhandler.LifecycleContext[AuditComponent, AuditScope, Output]) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if entity == nil || state.Previous == nil || state.Parent == nil {
		return fmt.Errorf("existing audit component is required")
	}
	if entity.Has == nil || !entity.Has.AuditEnabled || entity.UpdatedAt.IsZero() || entity.Etag != state.Previous.Etag {
		return fmt.Errorf("audit component update requires current revision, enabled flag and timestamp")
	}
	entity.SetEtag(entity.Etag + 1)
	return nil
}

// ToolRules customizes role Input.Scopes.ToolSettings.
type ToolRules struct{}

func ToolRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*ToolRules)(nil)).Elem()
}

var (
	ToolRulesHooks = new(ToolRules)
	ToolRulesDatly = ToolRulesDatlyType()
)

func (hooks *ToolRules) Init(ctx context.Context, entity *ToolSetting, state xhandler.LifecycleContext[ToolSetting, AuditScope, Output]) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if entity == nil || state.Parent == nil || entity.NamespaceId == nil || state.Parent.NamespaceId == nil || *entity.NamespaceId != *state.Parent.NamespaceId {
		return fmt.Errorf("audit tool must belong to namespace")
	}
	if entity.ToolName == nil || *entity.ToolName == "" || entity.UpdatedBy == nil || *entity.UpdatedBy == "" || entity.UpdatedAt.IsZero() || entity.Has == nil || !entity.Has.Enabled {
		return fmt.Errorf("audit tool update is incomplete")
	}
	if state.Previous == nil {
		entity.SetCreatedAt(entity.UpdatedAt)
		entity.SetCreatedBy(entity.UpdatedBy)
	}
	return nil
}
