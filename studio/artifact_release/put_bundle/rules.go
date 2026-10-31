package put_bundle

import (
	"context"
	releasevalue "github.com/viant/datly-studio/sdk/artifactrelease/value"
	contract "github.com/viant/forge/backend/artifactrelease"
	xhandler "github.com/viant/xdatly/handler"
	"strings"
)

type Rules struct {
	Input *Input `bind:"kind=input,required"`
}

func str(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func (rules *Rules) Init(ctx context.Context, row *NewBundle, state xhandler.LifecycleContext[NewBundle, xhandler.NoParent, Output]) error {
	if rules == nil || rules.Input == nil || row == nil {
		return contract.ErrInvalid
	}
	i := rules.Input
	scope := releasevalue.Scope{Tenant: str(row.TenantId), Account: str(row.AccountId), Workspace: str(row.WorkspaceId), Namespace: str(row.Namespace)}
	if !scope.Valid() || scope.ID() != i.ScopeId || str(row.ScopeId) != i.ScopeId || i.Actor == "" || strings.TrimSpace(i.Actor) != i.Actor || strings.ContainsRune(i.Actor, 0) || str(row.CreatedBy) != i.Actor || str(row.UpdatedBy) != i.Actor || row.CreatedAt == nil || row.UpdatedAt == nil || !row.CreatedAt.Equal(*row.UpdatedAt) {
		return contract.ErrInvalid
	}
	var b contract.Bundle
	if row.BundleJson == nil || releasevalue.StrictJSON(*row.BundleJson, &b) != nil || contract.ValidateBundle(b) != nil || b.Hash != str(row.BundleHash) || b.Namespace != scope.Namespace {
		return contract.ErrInvalid
	}
	return nil
}
func (*Rules) Validate(context.Context, *NewBundle, xhandler.LifecycleContext[NewBundle, xhandler.NoParent, Output]) error {
	return nil
}
