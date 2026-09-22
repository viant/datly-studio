package sqltransport

import (
	"context"
	"database/sql"
	"errors"

	"github.com/viant/datly-studio/sdk"
)

func (t *Transport) reportCapabilities(ctx context.Context, reportID string) (sdk.ReportCapabilities, error) {
	principal, ok := sdk.PrincipalFromContext(ctx)
	if !ok {
		return sdk.ReportCapabilities{CanView: true, CanRun: true, CanEdit: true, CanPublish: true, CanUseDQL: true}, nil
	}
	var owner string
	if err := t.DB.QueryRowContext(ctx, `SELECT owner_id FROM reports WHERE id=? AND deleted_at IS NULL`, reportID).Scan(&owner); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sdk.ReportCapabilities{}, &sdk.Error{Code: sdk.ErrorNotFound, Message: "report not found"}
		}
		return sdk.ReportCapabilities{}, internal(err)
	}
	if owner == principal.Subject {
		return sdk.ReportCapabilities{CanView: true, CanRun: true, CanEdit: true, CanPublish: true, CanUseDQL: true}, nil
	}
	result := sdk.ReportCapabilities{}
	err := t.DB.QueryRowContext(ctx, `SELECT can_view,can_run,can_edit,can_publish,can_use_dql FROM report_acl WHERE report_id=? AND subject_type='user' AND subject_id=?`, reportID, principal.Subject).
		Scan(&result.CanView, &result.CanRun, &result.CanEdit, &result.CanPublish, &result.CanUseDQL)
	if errors.Is(err, sql.ErrNoRows) {
		return result, nil
	}
	if err != nil {
		return sdk.ReportCapabilities{}, internal(err)
	}
	return result, nil
}
