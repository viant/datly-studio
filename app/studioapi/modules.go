package studioapi

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/viant/authz"
)

type Module struct {
	Path    string
	Handler http.Handler
}
type ModuleFactory func(context.Context, *sql.DB, authz.Provider) ([]Module, error)

func mountModules(mux *http.ServeMux, modules []Module) error {
	if mux == nil {
		return errors.New("Studio module mux required")
	}
	paths := map[string]bool{}
	for _, m := range modules {
		// Extensions cannot shadow Studio SDK/login/MCP routes or the asset root.
		if m.Handler == nil || !strings.HasPrefix(m.Path, "/api/") || strings.ContainsAny(m.Path, " ?#{}\\") || strings.Contains(m.Path, "//") || strings.Contains(m.Path, "/../") || strings.HasSuffix(m.Path, "/..") {
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
