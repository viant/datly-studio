package predicatepolicy

import (
	"github.com/viant/datly-studio/studio/settings"
	"github.com/viant/datly/spec"
	"path"
	"strings"
)

type RequiredError struct{}

func (*RequiredError) Error() string {
	return "component must define a required authorization predicate from its namespace predicate packages"
}

// ValidateDeclarations uses Datly's compiled parameter and import metadata.
// Optional source validation may additionally inspect application in effective SQL.
func ValidateDeclarations(component *spec.Component, policy *settings.Namespace) error {
	if policy == nil || policy.Security == nil || !policy.Security.RequireSecurityPredicate {
		return nil
	}
	if component == nil {
		return &RequiredError{}
	}
	selected := map[string]bool{}
	for _, pkg := range policy.Security.PredicatePackages {
		selected[pkg] = true
	}
	for _, parameter := range spec.EffectiveParameters(component.Parameters) {
		if parameter == nil {
			continue
		}
		for _, predicate := range parameter.Predicates {
			if predicate == nil || !strings.EqualFold(predicate.Name, "handler") || len(predicate.Args) == 0 {
				continue
			}
			if (parameter.Required == nil || !*parameter.Required) && !predicate.ApplyWhenAbsent {
				continue
			}
			identity := strings.TrimPrefix(strings.TrimSpace(predicate.Args[0]), "*")
			at := strings.LastIndex(identity, ".")
			if at < 1 {
				continue
			}
			pkg := identity[:at]
			if component.TypeContext != nil {
				for _, imp := range component.TypeContext.Imports {
					alias := imp.Alias
					if alias == "" {
						alias = path.Base(imp.Package)
					}
					if pkg == alias {
						pkg = imp.Package
						break
					}
				}
			}
			if selected[pkg] {
				return nil
			}
		}
	}
	return &RequiredError{}
}
