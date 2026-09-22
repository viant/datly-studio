// Package host is the public embedding boundary for Datly Studio.
// Applications extend Studio by supplying explicit predicate packages instead
// of modifying Studio's internal registries.
package host

import (
	"github.com/viant/datly-studio/studio/authorization"
	"github.com/viant/datly-studio/studio/predicatecatalog"
)

// Config contains host-owned Studio extensions. PredicatePackages are linked
// into the executable by the embedding application and become selectable in
// Studio's governed authorization-predicate catalog.
type Config struct {
	PredicatePackages []predicatecatalog.Package
}

// PredicateCatalog combines Studio's handlers with application handlers.
func (c Config) PredicateCatalog() (*predicatecatalog.Catalog, error) {
	packages := []predicatecatalog.Package{{
		Alias: "studioauthorization",
		Path:  "github.com/viant/datly-studio/studio/authorization",
		Types: authorization.DatlyPredicateHandlerTypes,
	}}
	packages = append(packages, c.PredicatePackages...)
	return predicatecatalog.New(packages...)
}
