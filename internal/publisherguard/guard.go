package publisherguard

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	guard "github.com/viant/datly-studio/studio/reports/store_global_access"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
	"github.com/viant/scy/auth/jwt"
	xhandler "github.com/viant/xdatly/handler"
	xresponse "github.com/viant/xdatly/response"
)

// AuthorizedInvoker binds the private global-publisher reader to the verified
// principal. The caller never supplies the guard's subject.
func AuthorizedInvoker(ctx context.Context, session xhandler.Session, claims *jwt.Claims, auth *studioauth.Output) (exec.ComponentInvoker, error) {
	if claims == nil || auth == nil || auth.Auth == nil || claims.Subject == "" || auth.Auth.Subject != claims.Subject {
		return nil, PublicError(403, "verified Studio principal is required")
	}
	if session == nil || session.Binder() == nil {
		return nil, fmt.Errorf("publisher guard session is required")
	}
	value, found, err := session.Binder().Lookup(ctx, exec.ComponentInvokerKey)
	if err != nil {
		return nil, err
	}
	invoker, ok := value.(exec.ComponentInvoker)
	if !found || !ok {
		return nil, fmt.Errorf("Datly component invoker is unavailable")
	}
	input := &guard.Input{}
	input.SetSubject(claims.Subject)
	value, err = invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{
		Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[guard.ReportComponent]().PkgPath(), Name: "report"},
		Route:     spec.RouteRef{Method: "GET", Path: "/_studio/report-store/global-access"},
	}, Input: input})
	if err != nil {
		return nil, err
	}
	allowed, ok := value.(*guard.Output)
	if !ok || allowed == nil {
		return nil, fmt.Errorf("global publisher guard returned %T", value)
	}
	if len(allowed.Reports) != 1 || allowed.Reports[0] == nil || allowed.Reports[0].Id == "" {
		return nil, PublicError(403, "publisher authorization is required")
	}
	return invoker, nil
}

func PublicError(code int, message string) error {
	return &xresponse.Error{Code: code, Cause: errors.New(message), Payload: map[string]string{"message": message}}
}
