package store_write

import (
	context "context"
	"fmt"
	xhandler "github.com/viant/xdatly/handler"
	reflect "reflect"
	"strings"
)

// NamespaceClaimRules customizes role Input.Claims.
type NamespaceClaimRules struct {
	Input *Input `bind:"kind=input,required"`
}

func NamespaceClaimRulesDatlyType() reflect.Type {
	return reflect.TypeOf((*NamespaceClaimRules)(nil)).Elem()
}

var (
	NamespaceClaimRulesHooks = new(NamespaceClaimRules)
	NamespaceClaimRulesDatly = NamespaceClaimRulesDatlyType()
)

func (hooks *NamespaceClaimRules) Init(_ context.Context, entity *StoredClaim, state xhandler.LifecycleContext[StoredClaim, xhandler.NoParent, Output]) error {
	if hooks.Input == nil || entity == nil || strings.TrimSpace(entity.Namespace) == "" ||
		strings.TrimSpace(entity.ReportId) == "" || entity.Has == nil ||
		!entity.Has.Namespace || !entity.Has.ReportId || !entity.Has.ShouldDelete {
		return fmt.Errorf("namespace claim requires exact namespace, report and operation marker")
	}
	previous := state.Previous
	if previous != nil && previous.ReportId != entity.ReportId {
		return &xhandler.Conflict{Entity: "resource_namespace_claim", Field: "report_id", Reason: "namespace belongs to another report"}
	}
	switch hooks.Input.Operation {
	case "acquire":
		if entity.ShouldDelete || !entity.Has.CreatedAt || !entity.Has.CreatedBy ||
			!entity.Has.UpdatedAt || !entity.Has.UpdatedBy || entity.CreatedAt.IsZero() ||
			entity.UpdatedAt.IsZero() || strings.TrimSpace(entity.CreatedBy) == "" ||
			strings.TrimSpace(entity.UpdatedBy) == "" {
			return fmt.Errorf("namespace acquisition requires complete audit fields")
		}
		if previous != nil {
			entity.SetCreatedAt(previous.CreatedAt)
			entity.SetCreatedBy(previous.CreatedBy)
		}
	case "release":
		if !entity.ShouldDelete || previous == nil {
			return &xhandler.Conflict{Entity: "resource_namespace_claim", Field: "namespace", Reason: "namespace claim is absent or not marked for release"}
		}
	default:
		return fmt.Errorf("unsupported namespace claim operation %q", hooks.Input.Operation)
	}
	return nil
}
func (hooks *NamespaceClaimRules) Validate(ctx context.Context, entity *StoredClaim, state xhandler.LifecycleContext[StoredClaim, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *NamespaceClaimRules) AfterSequence(ctx context.Context, entity *StoredClaim, state xhandler.LifecycleContext[StoredClaim, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *NamespaceClaimRules) AfterQueue(ctx context.Context, entity *StoredClaim, state xhandler.LifecycleContext[StoredClaim, xhandler.NoParent, Output]) error {
	return nil
}
func (hooks *NamespaceClaimRules) Finalize(ctx context.Context, input *Input, output *Output, outcome xhandler.Outcome) error {
	return nil
}
