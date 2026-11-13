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
	parameter := &spec.Parameter{Required: &required, Predicates: []*spec.Predicate{{Auth: true, Name: "handler", Args: []string{"security.Scope"}}}}
	component.Parameters = []*spec.Parameter{parameter}
	parameter.Predicates[0].Auth = false
	require.Error(t, predicatepolicy.ValidateDeclarations(component, alpha), "ordinary handler must not satisfy auth requirement")
	parameter.Predicates[0].Auth = true
	require.NoError(t, predicatepolicy.ValidateDeclarations(component, alpha))
	require.Error(t, predicatepolicy.ValidateDeclarations(component, beta))
	parameter.Required = nil
	require.Error(t, predicatepolicy.ValidateDeclarations(component, alpha))
	parameter.Predicates[0].ApplyWhenAbsent = true
	require.NoError(t, predicatepolicy.ValidateDeclarations(component, alpha))
	require.NoError(t, predicatepolicy.ValidateDeclarations(&spec.Component{}, &settings.Namespace{Security: &settings.Security{RequireSecurityPredicate: false}}))
}

func TestAuthMarkerIsOccurrenceSpecificAndDoesNotGrantAccessByItself(t *testing.T) {
	required := true
	policy := &settings.Namespace{Security: &settings.Security{RequireSecurityPredicate: true, PredicatePackages: []string{"example.com/security"}}}
	p := &spec.Parameter{Required: &required, Predicates: []*spec.Predicate{
		{Name: "handler", Args: []string{"example.com/security.Filter"}},
		{Name: "handler", Auth: true, Args: []string{"example.com/unapproved.Scope"}},
	}}
	c := &spec.Component{Parameters: []*spec.Parameter{p}}
	require.Error(t, predicatepolicy.ValidateDeclarations(c, policy))
	p.Predicates[0].Auth = true
	require.NoError(t, predicatepolicy.ValidateDeclarations(c, policy))
	p.Predicates[0].Auth = false
	policy.Security.RequireSecurityPredicate = false
	require.NoError(t, predicatepolicy.ValidateDeclarations(c, policy), "disabled namespace requirement stays disabled")
}
