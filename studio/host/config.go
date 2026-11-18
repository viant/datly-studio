// Package host is the public embedding boundary for Datly Studio.
// Applications extend Studio by supplying linked predicate and codec packages instead
// of modifying Studio's internal registries.
package host

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/viant/datly-studio/internal/warmupoperation"
	druntime "github.com/viant/datly/runtime"

	"github.com/viant/datly-studio/runtime/accesscontext"
	"github.com/viant/datly-studio/studio/authorization"
	"github.com/viant/datly-studio/studio/codeccatalog"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	"github.com/viant/datly/typecatalog"
	sqlxtypes "github.com/viant/sqlx/types"
	"github.com/viant/x"
)

const studioAuthorizationPath = "github.com/viant/datly-studio/studio/authorization"

// Config contains host-owned Studio extensions. PredicatePackages are linked
// into the executable by the embedding application and become selectable in
// Studio's governed authorization-predicate catalog.
type Config struct {
	// Observability is borrowed by dynamic execution runtimes. The supplying
	// host owns its lifetime; authoring snapshots never shut it down.
	Observability *druntime.Observability
	// WarmupOperationTimeout is a trusted duration string, for example "10m".
	// Nil retains the historical five-minute budget; invalid values fail startup.
	WarmupOperationTimeout *string
	PredicatePackages      []predicatecatalog.Package
	CodecPackages          []codeccatalog.Package
}

func (c Config) WarmupTimeout() (time.Duration, error) {
	return warmupoperation.Timeout(c.WarmupOperationTimeout)
}

// RuntimeTypes combines linked predicates with server-owned authorization
// dependency shapes for native authoring and dynamic readers.
func (c Config) RuntimeTypes() (*typecatalog.Catalog, error) {
	_, types, err := c.RuntimeCatalogs()
	return types, err
}

// RuntimeCatalogs captures predicate selection and runtime type authority together.
// Codecs are executable runtime types and never enter the predicate picker.
func (c Config) RuntimeCatalogs() (*predicatecatalog.Catalog, *typecatalog.Catalog, error) {
	predicates, err := c.PredicateCatalog()
	if err != nil {
		return nil, nil, err
	}
	types, err := predicates.RuntimeTypes()
	if err != nil {
		return nil, nil, err
	}
	if err = codeccatalog.Link(types, c.codecPackages()...); err != nil {
		return nil, nil, err
	}
	if err = codeccatalog.RegisterBuiltinTypes(types); err != nil {
		return nil, nil, err
	}
	// SQLX discovers MySQL BIT(1) as this named scanner type. Keep its
	// runtime identity in the same startup authority used by dynamic readers.
	if err = types.LinkRuntimeAll(typecatalog.TypeOriginPackage, x.NewType(reflect.TypeFor[sqlxtypes.BitBool]())); err != nil {
		return nil, nil, err
	}
	if err = accesscontext.RegisterTypes(types); err != nil {
		return nil, nil, err
	}
	return predicates, types, nil
}

// PredicateCatalog combines Studio's handlers with application handlers and
// the same trusted package-path allowlist across static, authoring, and
// dynamic hosts. Linked handler types are scanned inside those packages.
func (c Config) PredicateCatalog() (*predicatecatalog.Catalog, error) {
	packages := []predicatecatalog.Package{{
		Alias: "studioauthorization",
		Path:  studioAuthorizationPath,
		Types: authorization.DatlyPredicateHandlerTypes,
	}}
	explicit := map[string]bool{}
	for _, pkg := range c.PredicatePackages {
		explicit[strings.TrimSpace(pkg.Path)] = true
	}
	for _, pkg := range registeredPredicatePackages() {
		if !explicit[pkg.Path] {
			packages = append(packages, pkg)
		}
	}
	packages = append(packages, c.PredicatePackages...)
	seen := map[string]bool{}
	for _, pkg := range packages {
		seen[strings.TrimSpace(pkg.Path)] = true
	}
	raw := strings.TrimSpace(os.Getenv("STUDIO_PREDICATE_PACKAGES"))
	if raw != "" {
		for _, item := range strings.Split(raw, ",") {
			packagePath := strings.TrimSpace(item)
			if packagePath == "" {
				return nil, fmt.Errorf("STUDIO_PREDICATE_PACKAGES contains an empty package path")
			}
			if seen[packagePath] {
				continue
			}
			seen[packagePath] = true
			packages = append(packages, predicatecatalog.Package{Alias: path.Base(packagePath), Path: packagePath})
		}
	}
	if raw = strings.TrimSpace(os.Getenv("STUDIO_AUTH_PREDICATE_TYPES")); raw != "" {
		allowed := map[string][]string{}
		for _, item := range strings.Split(raw, ",") {
			packagePath, typeName, ok := strings.Cut(strings.TrimSpace(item), "#")
			packagePath, typeName = strings.TrimSpace(packagePath), strings.TrimSpace(typeName)
			if !ok || packagePath == "" || typeName == "" || packagePath == studioAuthorizationPath {
				return nil, fmt.Errorf("STUDIO_AUTH_PREDICATE_TYPES requires external package#Type entries")
			}
			allowed[packagePath] = append(allowed[packagePath], typeName)
		}
		for i := range packages {
			if packages[i].Path == studioAuthorizationPath {
				continue
			}
			packages[i].AuthorizationTypes = append([]string{}, allowed[packages[i].Path]...)
			delete(allowed, packages[i].Path)
		}
		for packagePath := range allowed {
			return nil, fmt.Errorf("STUDIO_AUTH_PREDICATE_TYPES package %s is not linked", packagePath)
		}
	}
	return predicatecatalog.New(packages...)
}
