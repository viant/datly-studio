package predicatepolicy_test

import (
	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/internal/predicatepolicy"
	"github.com/viant/datly-studio/studio/settings"
	"github.com/viant/datly/spec"
	"testing"
)

func TestRequiredAuthorizationDeclarationIsScopedByNamespacePackage(t *testing.T) {
	alpha := &settings.Namespace{Security: &settings.Security{PredicatePackages: []string{"example.com/security/alpha"}, RequireSecurityPredicate: true}}
	beta := &settings.Namespace{Security: &settings.Security{PredicatePackages: []string{"example.com/security/beta"}, RequireSecurityPredicate: true}}
	component := &spec.Component{TypeContext: &spec.TypeContext{Imports: []spec.ImportSpec{{Alias: "security", Package: "example.com/security/alpha"}}}}
	require.Error(t, predicatepolicy.ValidateDeclarations(component, alpha))
	required := true
	parameter := &spec.Parameter{Required: &required, Predicates: []*spec.Predicate{{Name: "handler", Args: []string{"security.Scope"}}}}
	component.Parameters = []*spec.Parameter{parameter}
	require.NoError(t, predicatepolicy.ValidateDeclarations(component, alpha))
	require.Error(t, predicatepolicy.ValidateDeclarations(component, beta))
	parameter.Required = nil
	require.Error(t, predicatepolicy.ValidateDeclarations(component, alpha))
	parameter.Predicates[0].ApplyWhenAbsent = true
	require.NoError(t, predicatepolicy.ValidateDeclarations(component, alpha))
	require.NoError(t, predicatepolicy.ValidateDeclarations(&spec.Component{}, &settings.Namespace{Security: &settings.Security{RequireSecurityPredicate: false}}))
}
