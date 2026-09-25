package authorization

import (
	"context"
	"database/sql"
	"strings"

	"github.com/viant/datly-studio/internal/reportcapability"
	"github.com/viant/datly-studio/sdk"
	sqltransport "github.com/viant/datly-studio/sdk/transport/sql"
)

// SDKAuthorizer applies Studio ownership and report ACL rules to the public
// SDK transport. Construct it with NewSDKAuthorizer for a reusable reader.
type SDKAuthorizer struct {
	DB                 *sql.DB
	ReportCapabilities *reportcapability.Reader
}

// NewSDKAuthorizer prepares the generated report-capability reader once for
// the lifetime of a serving host. The caller owns Close.
func NewSDKAuthorizer(db *sql.DB) (*SDKAuthorizer, error) {
	reader, err := reportcapability.New(db)
	if err != nil {
		return nil, err
	}
	return &SDKAuthorizer{DB: db, ReportCapabilities: reader}, nil
}

func (a *SDKAuthorizer) Close(ctx context.Context) error {
	if a == nil || a.ReportCapabilities == nil {
		return nil
	}
	return a.ReportCapabilities.Close(ctx)
}

func (a SDKAuthorizer) Authorize(ctx context.Context, request sqltransport.AuthorizationRequest) error {
	principal, ok := sdk.PrincipalFromContext(ctx)
	if !ok || a.DB == nil {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Studio principal is required"}
	}
	if request.OwnerID != "" && request.OwnerID != principal.Subject {
		return &sdk.Error{Code: sdk.ErrorForbidden, Message: "cannot create another principal's resource"}
	}
	if request.ReportID != "" {
		return a.report(ctx, principal.Subject, request.ReportID, request.Permission)
	}
	if request.ConnectorName != "" {
		return a.connector(ctx, principal.Subject, request.ConnectorName, request.Permission)
	}
	if request.NamespaceName != "" {
		return a.namespace(ctx, principal.Subject, request.NamespaceName, request.Permission)
	}
	if request.Permission == "publish" {
		return a.global(ctx, principal.Subject)
	}
	return nil
}

func (a SDKAuthorizer) namespace(ctx context.Context, subject, name, permission string) error {
	column := permissionColumn(permission)
	if permission == "edit" || permission == "publish" {
		return a.allow(ctx, `SELECT EXISTS(SELECT 1 FROM namespaces WHERE owner_id=? AND name=? AND deleted_at IS NULL)`, subject, name)
	}
	query := `SELECT EXISTS(SELECT 1 FROM namespaces n WHERE n.name=? AND n.deleted_at IS NULL AND (n.owner_id=? OR EXISTS (
SELECT 1 FROM reports r JOIN report_acl acl ON acl.report_id=r.id
WHERE r.owner_id=n.owner_id AND r.namespace=n.name AND r.deleted_at IS NULL
AND acl.subject_type='user' AND acl.subject_id=? AND acl.` + column + `=TRUE)))`
	return a.allow(ctx, query, name, subject, subject)
}

func (a SDKAuthorizer) report(ctx context.Context, subject, id, permission string) error {
	reader := a.ReportCapabilities
	owned := false
	if reader == nil {
		var err error
		reader, err = reportcapability.New(a.DB)
		if err != nil {
			return denied()
		}
		owned = true
	}
	row, readErr := reader.Read(ctx, id, subject)
	var closeErr error
	if owned {
		closeErr = reader.Close(context.Background())
	}
	if readErr != nil || closeErr != nil || row == nil {
		return denied()
	}
	if row.OwnerId == subject {
		return nil
	}
	switch permissionColumn(permission) {
	case "can_run":
		if row.CanRun {
			return nil
		}
	case permissionEdit:
		if row.CanEdit {
			return nil
		}
	case permissionPublish:
		if row.CanPublish {
			return nil
		}
	case "can_use_dql":
		if row.CanUseDql {
			return nil
		}
	default:
		if row.CanView {
			return nil
		}
	}
	return denied()
}

func (a SDKAuthorizer) connector(ctx context.Context, subject, name, permission string) error {
	column := permissionColumn(permission)
	query := `SELECT EXISTS(SELECT 1 FROM connectors c WHERE c.name=? AND c.deleted_at IS NULL AND (c.owner_id=? OR EXISTS (SELECT 1 FROM reports r JOIN report_acl acl ON acl.report_id=r.id WHERE r.default_connector_name=c.name AND r.deleted_at IS NULL AND acl.subject_type='user' AND acl.subject_id=? AND acl.` + column + `=TRUE)))`
	return a.allow(ctx, query, name, subject, subject)
}

func (a SDKAuthorizer) global(ctx context.Context, subject string) error {
	return a.allow(ctx, `SELECT EXISTS(SELECT 1 FROM reports WHERE owner_id=? AND deleted_at IS NULL)
OR EXISTS(SELECT 1 FROM report_acl WHERE subject_type='user' AND subject_id=? AND can_publish=TRUE)`, subject, subject)
}

func (a SDKAuthorizer) allow(ctx context.Context, query string, args ...any) error {
	var allowed bool
	if err := a.DB.QueryRowContext(ctx, query, args...).Scan(&allowed); err != nil || !allowed {
		return denied()
	}
	return nil
}

func denied() error {
	return &sdk.Error{Code: sdk.ErrorForbidden, Message: "Studio authorization denied"}
}

func permissionColumn(permission string) string {
	switch strings.TrimSpace(permission) {
	case "publish":
		return permissionPublish
	case "edit":
		return permissionEdit
	case "run":
		return "can_run"
	case "dql":
		return "can_use_dql"
	default:
		return permissionView
	}
}
