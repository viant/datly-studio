package sqltransport

import (
	"context"
	"database/sql"
	"errors"

	"github.com/viant/datly-studio/sdk"
)

func (t *Transport) reportCapabilities(ctx context.Context, reportID string) (caps sdk.ReportCapabilities, err error) {
	defer func() { caps.CanManageACL = err == nil && caps.CanPublish && t.aclAvailable(ctx) }()
	principal, ok := sdk.PrincipalFromContext(ctx)
	if !ok {
		return sdk.ReportCapabilities{CanView: true, CanRun: true, CanEdit: true, CanPublish: true, CanUseDQL: true}, nil
	}
	row, readErr := t.readReportCapability(ctx, reportID, principal.Subject)
	if readErr != nil {
		if errors.Is(readErr, sql.ErrNoRows) {
			return sdk.ReportCapabilities{}, &sdk.Error{Code: sdk.ErrorNotFound, Message: "report not found"}
		}
		return sdk.ReportCapabilities{}, internal(readErr)
	}
	if row.OwnerId == principal.Subject {
		return sdk.ReportCapabilities{CanView: true, CanRun: true, CanEdit: true, CanPublish: true, CanUseDQL: true}, nil
	}
	return sdk.ReportCapabilities{CanView: row.CanView, CanRun: row.CanRun, CanEdit: row.CanEdit,
		CanPublish: row.CanPublish, CanUseDQL: row.CanUseDql}, nil
}

func (t *Transport) aclAvailable(ctx context.Context) bool {
	principal, ok := sdk.PrincipalFromContext(ctx)
	if !ok || principal.Development {
		return false
	}
	return true
}
