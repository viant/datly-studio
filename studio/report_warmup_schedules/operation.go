// Package report_warmup_schedules shares the existing publish-authorized native
// entry path for exact-version schedule management.
package report_warmup_schedules

import (
	"context"
	"fmt"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly-studio/internal/publisherguard"
	"github.com/viant/datly-studio/internal/versionrun"
	"github.com/viant/datly-studio/sdk"
	sqltransport "github.com/viant/datly-studio/sdk/transport/sql"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	guard "github.com/viant/datly-studio/studio/reports/publish_guard"
	"github.com/viant/datly/exec"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/spec"
	"github.com/viant/scy/auth/jwt"
	xconnector "github.com/viant/xdatly/connector"
	xhandler "github.com/viant/xdatly/handler"
	"reflect"
)

func Manage(ctx context.Context, session xhandler.Session, claims *jwt.Claims, auth *studioauth.Output, report string, version int, operation string, input, output any) error {
	if claims == nil || auth == nil || auth.Auth == nil || claims.Subject == "" || claims.Subject != auth.Auth.Subject {
		return publisherguard.PublicError(403, "verified Studio principal is required")
	}
	if session == nil || session.Binder() == nil {
		return fmt.Errorf("native schedule session is required")
	}
	value, found, e := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if e != nil {
		return e
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return fmt.Errorf("native component invoker is unavailable")
	}
	access := &guard.Input{}
	access.SetJwt(claims)
	access.SetAuth(auth)
	access.SetReportId(report)
	value, e = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[guard.ReportComponent]().PkgPath(), Name: "report"}, Route: spec.RouteRef{Method: "POST", Path: "/_studio/reports/publish-guard"}}, Input: access})
	if e != nil {
		return e
	}
	allowed, ok := value.(*guard.Output)
	if !ok || allowed == nil || allowed.Item == nil || allowed.Item.Id != report {
		return publisherguard.PublicError(403, "warmup publish access is required")
	}
	value, found, e = session.Binder().Lookup(ctx, rhandler.ConnectorCapabilityKey)
	if e != nil {
		return e
	}
	provider, ok := value.(xconnector.Provider)
	if !found || !ok {
		return fmt.Errorf("trusted Studio connector is unavailable")
	}
	db, e := provider.Connector(ctx, "studio")
	if e != nil || db == nil {
		return fmt.Errorf("configured Studio database is unavailable")
	}
	ctx = sdk.WithPrincipal(ctx, sdk.Principal{Subject: claims.Subject})
	ctx = sdk.WithNamespaceSelection(ctx, namespaceaccess.ID(allowed.Item.OwnerId, allowed.Item.Namespace))
	engine := &sqltransport.Transport{DB: db, ComponentInvoker: invoker}
	if e = engine.ManageWarmupSchedule(ctx, operation, input, output); e != nil {
		return versionrun.PublicError(e, "warmup schedule")
	}
	return nil
}
