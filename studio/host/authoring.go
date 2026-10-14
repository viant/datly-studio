package host

import (
	"context"
	"net/http"
	"strings"

	"github.com/viant/datly-studio/runtime/accesscontext"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	"github.com/viant/datly/typecatalog"
	"golang.org/x/mod/module"
)

const DefaultAuthoringModule = "github.com/viant/datly-studio"

// AuthoringConfig is trusted startup configuration, never a request parameter.
type AuthoringConfig struct {
	ModulePath string
	Config     Config
}

// Authoring is a detached immutable per-host compilation snapshot.
type Authoring struct {
	module     string
	predicates *predicatecatalog.Catalog
	types      *typecatalog.Catalog
}

func NewAuthoring(config AuthoringConfig) (*Authoring, error) {
	name := strings.TrimSpace(config.ModulePath)
	if name == "" {
		name = DefaultAuthoringModule
	}
	if err := module.CheckPath(name); err != nil {
		return nil, err
	}
	// Catalog construction captures the supplied slices and environment once.
	predicates, err := config.Config.PredicateCatalog()
	if err != nil {
		return nil, err
	}
	types, err := predicates.RuntimeTypes()
	if err != nil {
		return nil, err
	}
	if err := accesscontext.RegisterTypes(types); err != nil {
		return nil, err
	}
	return &Authoring{module: name, predicates: predicates, types: types}, nil
}

func (a *Authoring) ModulePath() string                          { return a.module }
func (a *Authoring) RuntimeTypes() (*typecatalog.Catalog, error) { return a.types.Clone() }
func (a *Authoring) PredicateCatalog() *predicatecatalog.Catalog { return a.predicates }

type authoringKey struct{}

// AuthoringFromContext preserves the standalone default when no host supplied a snapshot.
func AuthoringFromContext(ctx context.Context) (*Authoring, error) {
	if value, ok := ctx.Value(authoringKey{}).(*Authoring); ok {
		return value, nil
	}
	return NewAuthoring(AuthoringConfig{})
}

func (a *Authoring) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), authoringKey{}, a)))
	})
}
