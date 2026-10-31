package sqltransport

import (
	"context"
	"github.com/viant/datly-studio/internal/predicatepolicy"
	"github.com/viant/datly-studio/internal/settingstore"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly/authoring/readerbuilder"
)

// Recheck the owning namespace at publication, even when source validation is older.
func (t *Transport) requireNamespaceAuthorizationPredicate(ctx context.Context, id string, version *sdk.ReportVersion) error {
	component, err := t.getReportValue(ctx, id)
	if err != nil {
		return err
	}
	policy, err := settingstore.Read(ctx, t.DB, component.OwnerID, component.Namespace)
	if err != nil {
		return err
	}
	if policy.Security == nil || !policy.Security.RequireSecurityPredicate {
		return nil
	}
	inspected, err := t.runReaderBuilder(ctx, id, version, readerbuilder.Operation{Type: readerbuilder.OperationInspect})
	if err != nil {
		return err
	}
	if inspected == nil || inspected.Structure == nil {
		return invalid(&predicatepolicy.RequiredError{})
	}
	if err = predicatepolicy.ValidateDeclarations(inspected.Structure.Component, policy); err != nil {
		return invalid(err)
	}
	return nil
}
