package studioapi

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/sdk/httptransport"
)

type Module struct {
	Path    string
	Handler http.Handler
}

// Module handlers may implement Close(context.Context) error. Serve waits for
// those optional process-owned workers before releasing its borrowed database.
// A shared handler mounted at multiple paths is closed only once.
func closeModules(ctx context.Context, modules []Module) error {
	seen := map[any]bool{}
	var result error
	for i := len(modules) - 1; i >= 0; i-- {
		handler := modules[i].Handler
		closer, ok := handler.(interface{ Close(context.Context) error })
		if !ok {
			continue
		}
		if reflect.TypeOf(handler).Comparable() {
			if seen[handler] {
				continue
			}
			seen[handler] = true
		}
		result = errors.Join(result, closer.Close(ctx))
	}
	return result
}

type ModuleFactory func(context.Context, *sql.DB, authz.Provider) ([]Module, error)

// AuthenticatedModuleFactory installs process-owned cookie-authenticated
// extensions after the BFF session runtime exists. Authenticate yields only
// server-side verified principal/credential context; the module must enforce
// its own resource authority. Nil keeps standalone routing unchanged.
type AuthenticatedModuleFactory func(context.Context, *sql.DB, authz.Provider, httptransport.Authenticator) ([]Module, error)

func mountModules(mux *http.ServeMux, modules []Module) error {
	if mux == nil {
		return errors.New("Studio module mux required")
	}
	paths := map[string]bool{}
	for _, m := range modules {
		// Extensions cannot shadow Studio SDK/login/MCP routes or the asset root.
		if m.Handler == nil || !(strings.HasPrefix(m.Path, "/api/") || m.Path == "/v1/artifacts/") || strings.ContainsAny(m.Path, " ?#{}\\") || strings.Contains(m.Path, "//") || strings.Contains(m.Path, "/../") || strings.HasSuffix(m.Path, "/..") {
			return errors.New("invalid Studio extension module mount")
		}
		if paths[m.Path] {
			return errors.New("duplicate Studio extension module mount")
		}
		paths[m.Path] = true
	}
	for _, m := range modules {
		mux.Handle(m.Path, m.Handler)
	}
	return nil
}

// corsModules permits non-browser bearer clients only on explicitly mounted,
// independently authenticated modules. Cookie-only SDK/browser requests retain
// the Origin requirement, and explicit foreign origins are always rejected.
func corsModules(origin string, required bool, modules []Module, next http.Handler, bearerPaths ...string) http.Handler {
	paths := make([]string, len(modules))
	for i, module := range modules {
		paths[i] = module.Path
	}
	paths = append(paths, bearerPaths...)
	return corsWithOriginException(origin, required, func(r *http.Request) bool {
		fields := strings.Fields(r.Header.Get("Authorization"))
		if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") || fields[1] == "" {
			return false
		}
		for _, path := range paths {
			if r.URL.Path == path || strings.HasSuffix(path, "/") && strings.HasPrefix(r.URL.Path, path) {
				return true
			}
		}
		return false
	}, next)
}
