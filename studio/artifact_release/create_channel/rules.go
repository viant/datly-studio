package create_channel

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
func (rules *Rules) Init(ctx context.Context, row *NewChannel, state xhandler.LifecycleContext[NewChannel, xhandler.NoParent, Output]) error {
	if rules == nil || rules.Input == nil || row == nil {
		return contract.ErrInvalid
	}
	i := rules.Input
	scope := releasevalue.Scope{Tenant: str(row.TenantId), Account: str(row.AccountId), Workspace: str(row.WorkspaceId), Namespace: str(row.Namespace)}
	if !scope.Valid() || scope.ID() != i.ScopeId || str(row.ScopeId) != i.ScopeId || i.Actor == "" || strings.TrimSpace(i.Actor) != i.Actor || strings.ContainsRune(i.Actor, 0) || str(row.CreatedBy) != i.Actor || str(row.UpdatedBy) != i.Actor || row.CreatedAt == nil || row.UpdatedAt == nil || !row.CreatedAt.Equal(*row.UpdatedAt) {
		return contract.ErrInvalid
	}
	c := contract.Channel{Key: contract.Key{Tenant: scope.Tenant, Account: scope.Account, Workspace: scope.Workspace, Namespace: scope.Namespace, Name: str(row.Name)}, Description: str(row.Description)}
	if row.Revision != 0 || row.PendingOperationId != nil || row.ActiveBundleHash != nil || row.ActiveDependenciesJson != nil || row.PolicyRefsJson == nil || releasevalue.StrictJSON(*row.PolicyRefsJson, &c.Policies) != nil || contract.ValidateChannel(c) != nil {
		return contract.ErrInvalid
	}
	return nil
}
func (*Rules) Validate(context.Context, *NewChannel, xhandler.LifecycleContext[NewChannel, xhandler.NoParent, Output]) error {
	return nil
}
