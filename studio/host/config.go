// Package host is the public embedding boundary for Datly Studio.
// Applications extend Studio by supplying explicit predicate packages instead
// of modifying Studio's internal registries.
package host

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/viant/datly-studio/runtime/accesscontext"
	"github.com/viant/datly-studio/studio/authorization"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	"github.com/viant/datly/typecatalog"
)

const studioAuthorizationPath = "github.com/viant/datly-studio/studio/authorization"

// Config contains host-owned Studio extensions. PredicatePackages are linked
// into the executable by the embedding application and become selectable in
// Studio's governed authorization-predicate catalog.
type Config struct {
	PredicatePackages []predicatecatalog.Package
}

// RuntimeTypes combines linked predicates with server-owned authorization
// dependency shapes for native authoring and dynamic readers.
func (c Config) RuntimeTypes() (*typecatalog.Catalog, error) {
	predicates, err := c.PredicateCatalog()
	if err != nil {
		return nil, err
	}
	types, err := predicates.RuntimeTypes()
	if err != nil {
		return nil, err
	}
	if err = accesscontext.RegisterTypes(types); err != nil {
		return nil, err
	}
	return types, nil
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
