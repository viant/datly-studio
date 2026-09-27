// Package host is the public embedding boundary for Datly Studio.
// Applications extend Studio by supplying explicit predicate packages instead
// of modifying Studio's internal registries.
package host

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/viant/datly-studio/studio/authorization"
	"github.com/viant/datly-studio/studio/predicatecatalog"
)

const studioAuthorizationPath = "github.com/viant/datly-studio/studio/authorization"

// Config contains host-owned Studio extensions. PredicatePackages are linked
// into the executable by the embedding application and become selectable in
// Studio's governed authorization-predicate catalog.
type Config struct {
	PredicatePackages []predicatecatalog.Package
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
	return predicatecatalog.New(packages...)
}
