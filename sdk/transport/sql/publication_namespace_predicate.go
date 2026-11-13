package sqltransport

import (
	"context"
	"github.com/viant/datly-studio/internal/predicatepolicy"
	"github.com/viant/datly-studio/internal/settingstore"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/authoring/readerbuilder"
)

// Publication preserves accepted source validation. Only exposure policy is live;
// changing authoring requirements cannot invalidate an immutable accepted revision.
func (t *Transport) requireNamespaceAuthorizationPredicate(ctx context.Context, id string, version *sdk.ReportVersion) error {
	return t.checkNamespaceDeclarations(ctx, id, version, false)
}

func (t *Transport) validateNamespaceAuthorizationPredicate(ctx context.Context, id string, version *sdk.ReportVersion) error {
	return t.checkNamespaceDeclarations(ctx, id, version, true)
}

func (t *Transport) checkNamespaceDeclarations(ctx context.Context, id string, version *sdk.ReportVersion, authoring bool) error {
	component, err := t.getReportValue(ctx, id)
	if err != nil {
		return err
	}
	policy, err := settingstore.Read(ctx, t.DB, component.OwnerID, component.Namespace)
	if err != nil {
		return err
	}
	if policy.Security == nil || (!policy.Security.RequireInternalComponents && (!authoring || !policy.Security.RequireSecurityPredicate)) {
		return nil
	}
	inspected, err := t.runReaderBuilder(ctx, id, version, readerbuilder.Operation{Type: readerbuilder.OperationInspect})
	if err != nil {
		return err
	}
	validate := predicatepolicy.ValidateInternalComponents
	if authoring {
		validate = predicatepolicy.ValidateDeclarations
	}
	if inspected == nil || inspected.Structure == nil {
		return invalid(validate(nil, policy))
	}
	if err = validate(inspected.Structure.Component, policy); err != nil {
		return invalid(err)
	}
	return nil
}
