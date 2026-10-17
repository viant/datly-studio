package host

import (
	"context"
	"github.com/viant/datly-studio/internal/connectorsecret"
	"github.com/viant/datly/constant"
	"net/http"
	"strings"

	"github.com/viant/datly-studio/runtime/accesscontext"
	"github.com/viant/datly-studio/runtime/preview"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/typecatalog"
	"golang.org/x/mod/module"
)

const DefaultAuthoringModule = "github.com/viant/datly-studio"

// AuthoringConfig is trusted startup configuration, never a request parameter.
type AuthoringConfig struct {
	Const        *constant.Values
	Dependencies preview.DependencyResolver
	ReaderCaches map[string]*spec.CacheSettings
	ModulePath   string
	Config       Config
}

// Authoring is a detached immutable per-host compilation snapshot.
type Authoring struct {
	constants    *constant.Values
	dependencies preview.DependencyResolver
	readerCaches map[string]*spec.CacheSettings
	module       string
	predicates   *predicatecatalog.Catalog
	types        *typecatalog.Catalog
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
	return &Authoring{constants: config.Const, module: name, predicates: predicates, types: types, readerCaches: cloneReaderCaches(config.ReaderCaches), dependencies: config.Dependencies}, nil
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
		next.ServeHTTP(w, r.WithContext(connectorsecret.WithConstants(context.WithValue(r.Context(), authoringKey{}, a), a.constants)))
	})
}

// ReaderCaches returns detached deployment settings for version validation and
// preview. Browser input cannot replace this authoring snapshot.
func (a *Authoring) ReaderCaches() map[string]*spec.CacheSettings {
	return cloneReaderCaches(a.readerCaches)
}
func cloneReaderCaches(source map[string]*spec.CacheSettings) map[string]*spec.CacheSettings {
	if source == nil {
		return nil
	}
	result := make(map[string]*spec.CacheSettings, len(source))
	for name, value := range source {
		result[name] = value.Clone()
	}
	return result
}

func (a *Authoring) Dependencies() preview.DependencyResolver { return a.dependencies }

// Constants returns immutable deployment values; request input cannot replace them.
func (a *Authoring) Constants() *constant.Values { return a.constants }
