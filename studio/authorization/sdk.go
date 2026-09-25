package authorization

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/viant/datly-studio/internal/connectoraccess"
	"github.com/viant/datly-studio/internal/globalaccess"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/reportcapability"
	"github.com/viant/datly-studio/sdk"
	sqltransport "github.com/viant/datly-studio/sdk/transport/sql"
)

// SDKAuthorizer applies Studio ownership and report ACL rules to the public
// SDK transport. Construct it with NewSDKAuthorizer for a reusable reader.
type SDKAuthorizer struct {
	DB                 *sql.DB
	ReportCapabilities *reportcapability.Reader
	NamespaceAccess    *namespaceaccess.Reader
	ConnectorAccess    *connectoraccess.Reader
	GlobalAccess       *globalaccess.Reader
}

// NewSDKAuthorizer prepares the generated report-capability reader once for
// the lifetime of a serving host. The caller owns Close.
func NewSDKAuthorizer(db *sql.DB) (*SDKAuthorizer, error) {
	reader, err := reportcapability.New(db)
	if err != nil {
		return nil, err
	}
	namespaceReader, err := namespaceaccess.New(db)
	if err != nil {
		_ = reader.Close(context.Background())
		return nil, err
	}
	connectorReader, err := connectoraccess.New(db)
	if err != nil {
		_ = errors.Join(reader.Close(context.Background()), namespaceReader.Close(context.Background()))
		return nil, err
	}
	globalReader, err := globalaccess.New(db)
	if err != nil {
		_ = errors.Join(reader.Close(context.Background()), namespaceReader.Close(context.Background()), connectorReader.Close(context.Background()))
		return nil, err
	}
	return &SDKAuthorizer{DB: db, ReportCapabilities: reader, NamespaceAccess: namespaceReader,
		ConnectorAccess: connectorReader, GlobalAccess: globalReader}, nil
}

func (a *SDKAuthorizer) Close(ctx context.Context) error {
	if a == nil {
		return nil
	}
	return errors.Join(a.ReportCapabilities.Close(ctx), a.NamespaceAccess.Close(ctx),
		a.ConnectorAccess.Close(ctx), a.GlobalAccess.Close(ctx))
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
	reader := a.NamespaceAccess
	owned := false
	if reader == nil {
		var err error
		reader, err = namespaceaccess.New(a.DB)
		if err != nil {
			return denied()
		}
		owned = true
	}
	allowed, readErr := reader.Allowed(ctx, subject, name, permission)
	var closeErr error
	if owned {
		closeErr = reader.Close(context.Background())
	}
	if readErr != nil || closeErr != nil || !allowed {
		return denied()
	}
	return nil
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
	reader := a.ConnectorAccess
	owned := false
	if reader == nil {
		var err error
		reader, err = connectoraccess.New(a.DB)
		if err != nil {
			return denied()
		}
		owned = true
	}
	allowed, readErr := reader.Allowed(ctx, subject, name, permission)
	var closeErr error
	if owned {
		closeErr = reader.Close(context.Background())
	}
	if readErr != nil || closeErr != nil || !allowed {
		return denied()
	}
	return nil
}

func (a SDKAuthorizer) global(ctx context.Context, subject string) error {
	reader := a.GlobalAccess
	owned := false
	if reader == nil {
		var err error
		reader, err = globalaccess.New(a.DB)
		if err != nil {
			return denied()
		}
		owned = true
	}
	allowed, readErr := reader.Allowed(ctx, subject)
	var closeErr error
	if owned {
		closeErr = reader.Close(context.Background())
	}
	if readErr != nil || closeErr != nil || !allowed {
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
