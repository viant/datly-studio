package predicatecatalog

import (
	"fmt"
	"github.com/viant/datly-studio/studio/settings"
)

func (c *Catalog) ValidateSecuritySettings(value *settings.Security) error {
	if value == nil {
		return nil
	}
	available := map[string]int{}
	for _, d := range c.Descriptors() {
		if d.Package != "github.com/viant/datly-studio/studio/authorization" {
			available[d.Package]++
		}
	}
	eligible := 0
	for _, pkg := range value.PredicatePackages {
		count := available[pkg]
		if count == 0 {
			return fmt.Errorf("predicate package %q has no host-approved security handlers", pkg)
		}
		eligible += count
	}

	if value.RequireSecurityPredicate && eligible == 0 {
		return fmt.Errorf("namespace requires at least one host-approved security predicate")
	}
	return nil
}
