package versionrun

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	access "github.com/viant/authz"
	accessstore "github.com/viant/authz/component/store/sql"
	"github.com/viant/datly-studio/internal/publisherguard"
	"github.com/viant/datly-studio/runtime/accessprovider"
	"github.com/viant/datly-studio/runtime/preview"
	"github.com/viant/datly-studio/sdk"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	"github.com/viant/datly-studio/studio/host"
	runaccess "github.com/viant/datly-studio/studio/reports/store_run_access"
	"github.com/viant/datly/exec"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/spec"
	"github.com/viant/scy/auth/jwt"
	xconnector "github.com/viant/xdatly/connector"
	xhandler "github.com/viant/xdatly/handler"
)

const modulePath = "github.com/viant/datly-studio"

// Authorized returns the exact-version Datly runtime after binding the
// verified principal and private report can_run decision. The borrowed Studio
// database remains owned by the static host; callers cancel the returned
// execution context but never close its DB.
func Authorized(ctx context.Context, session xhandler.Session, claims *jwt.Claims, auth *studioauth.Output,
	reportID string, versionNo int, timeout time.Duration) (context.Context, preview.Dynamic, context.CancelFunc, error) {
	if claims == nil || auth == nil || auth.Auth == nil || claims.Subject == "" || auth.Auth.Subject != claims.Subject {
		return nil, preview.Dynamic{}, nil, publisherguard.PublicError(403, "verified Studio principal is required")
	}
	if strings.TrimSpace(reportID) == "" || versionNo <= 0 {
		return nil, preview.Dynamic{}, nil, publisherguard.PublicError(400, "reportId and positive versionNo are required")
	}
	if session == nil || session.Binder() == nil {
		return nil, preview.Dynamic{}, nil, fmt.Errorf("reader execution session is required")
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return nil, preview.Dynamic{}, nil, err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return nil, preview.Dynamic{}, nil, fmt.Errorf("Datly component invoker is unavailable")
	}
	guard := &runaccess.Input{}
	guard.SetReportId(reportID)
	guard.SetSubject(claims.Subject)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{
		Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[runaccess.AccessComponent]().PkgPath(), Name: "access"},
		Route:     spec.RouteRef{Method: "GET", Path: "/_studio/report-run-access"},
	}, Input: guard})
	if err != nil {
		return nil, preview.Dynamic{}, nil, err
	}
	allowed, ok := value.(*runaccess.Output)
	if !ok || allowed == nil || len(allowed.Allowed) != 1 || allowed.Allowed[0] == nil || allowed.Allowed[0].ReportId != reportID {
		return nil, preview.Dynamic{}, nil, publisherguard.PublicError(403, "report run access is required")
	}
	value, found, err = session.Binder().Lookup(ctx, rhandler.ConnectorCapabilityKey)
	if err != nil {
		return nil, preview.Dynamic{}, nil, err
	}
	provider, ok := value.(xconnector.Provider)
	if !found || !ok {
		return nil, preview.Dynamic{}, nil, fmt.Errorf("trusted Studio connector capability is unavailable")
	}
	db, err := provider.Connector(ctx, "studio")
	if err != nil || db == nil {
		return nil, preview.Dynamic{}, nil, fmt.Errorf("configured Studio database is unavailable")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	types, err := (host.Config{}).RuntimeTypes()
	if err != nil {
		return nil, preview.Dynamic{}, nil, err
	}
	executionCtx, cancel := context.WithTimeout(ctx, timeout)
	executionCtx = sdk.WithPrincipal(executionCtx, sdk.Principal{Subject: claims.Subject})
	engine := preview.Dynamic{StudioDB: db, ModulePath: modulePath, Types: types}
	aclProvider, err := accessprovider.FromEnvironment(executionCtx)
	if err != nil {
		cancel()
		return nil, preview.Dynamic{}, nil, publisherguard.PublicError(503, "ACL verifier is not configured correctly")
	}
	credential := sdk.VerifiedCredential{Claims: claims}
	if aclProvider != nil {
		var header struct {
			Authorization string `bind:"kind=header,in=Authorization,required"`
		}
		if err := session.Binder().Bind(ctx, &header); err != nil || !strings.HasPrefix(header.Authorization, "Bearer ") || strings.TrimSpace(strings.TrimPrefix(header.Authorization, "Bearer ")) == "" {
			cancel()
			return nil, preview.Dynamic{}, nil, publisherguard.PublicError(401, "ACL bearer credential is required")
		}
		credential.Bearer = strings.TrimSpace(strings.TrimPrefix(header.Authorization, "Bearer "))
		engine.Access = &access.Service{Store: &accessstore.Store{DB: db, Invoker: invoker}, Provider: aclProvider}
		engine.AccessTenant = strings.TrimSpace(os.Getenv("STUDIO_ACCESS_TENANT"))
	}
	executionCtx = sdk.WithVerifiedCredential(executionCtx, credential)
	return executionCtx, engine, cancel, nil
}
