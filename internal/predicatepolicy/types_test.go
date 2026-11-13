package predicatepolicy_test

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/examples/predicates"
	"github.com/viant/datly-studio/internal/predicatepolicy"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	"github.com/viant/datly-studio/studio/settings"
	rangefilter "github.com/viant/datly/runtime/predicate"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/typecatalog"
	x "github.com/viant/x"
	xpredicate "github.com/viant/xdatly/predicate"
	"reflect"
	"testing"
)

func TestNamespaceSelectionRetainsOtherTypesAndRemovesUnselectedPredicates(t *testing.T) {
	source := typecatalog.NewCatalog()
	predicate := x.NewType(reflect.TypeFor[predicates.OrderPresence]())
	type dependency struct{ Value int }
	shape := x.NewType(reflect.TypeFor[dependency]())
	require.NoError(t, source.RegisterAll(typecatalog.TypeOriginPackage, predicate, shape))
	selected, err := predicatepolicy.Types(source, &settings.Namespace{Security: &settings.Security{PredicatePackages: []string{}}})
	require.NoError(t, err)
	_, found, err := selected.Resolve(typecatalog.PackageAuthority, predicate.Key())
	require.NoError(t, err)
	require.False(t, found)
	_, found, err = selected.Resolve(typecatalog.PackageAuthority, shape.Key())
	require.NoError(t, err)
	require.True(t, found)
	inherited, err := predicatepolicy.Types(source, &settings.Namespace{Security: &settings.Security{PredicatePackages: nil}})
	require.NoError(t, err)
	_, found, err = inherited.Resolve(typecatalog.PackageAuthority, predicate.Key())
	require.NoError(t, err)
	require.False(t, found)
	// Missing configuration never opts a package in; selection never mutates the host catalog.
	_, found, err = source.Resolve(typecatalog.PackageAuthority, predicate.Key())
	require.NoError(t, err)
	require.True(t, found)
}

func TestDiscoveryShowsOnlyNamespaceSelectedPackages(t *testing.T) {
	internal := "github.com/viant/datly-studio/studio/authorization"
	allowed := "example.com/predicates"
	policy := &settings.Namespace{Security: &settings.Security{PredicatePackages: []string{allowed}}}
	require.True(t, predicatepolicy.AllowsPackage(policy, allowed))
	require.False(t, predicatepolicy.AllowsPackage(policy, internal), "internal compilation support is not an authoring choice")
	require.False(t, predicatepolicy.AllowsPackage(policy, "example.com/other"))
	require.False(t, predicatepolicy.AllowsPackage(&settings.Namespace{}, allowed))
	require.True(t, predicatepolicy.AllowsPackage(nil, allowed), "global administration retains global discovery")
}

type configuredPredicate struct{}

func (*configuredPredicate) Compute(context.Context, any, ...any) (*xpredicate.Criteria, error) {
	return &xpredicate.Criteria{Expression: "1=0"}, nil
}
func TestNamespaceSelectionAlsoFiltersConfiguredHandlers(t *testing.T) {
	source := typecatalog.NewCatalog()
	predicate := x.NewType(reflect.TypeFor[configuredPredicate]())
	require.NoError(t, source.RegisterAll(typecatalog.TypeOriginPackage, predicate))
	filtered, err := predicatepolicy.Types(source, &settings.Namespace{})
	require.NoError(t, err)
	_, found, err := filtered.Resolve(typecatalog.PackageAuthority, predicate.Key())
	require.NoError(t, err)
	require.False(t, found)
	selected, err := predicatepolicy.Types(source, &settings.Namespace{Security: &settings.Security{PredicatePackages: []string{predicate.PkgPath}}})
	require.NoError(t, err)
	_, found, err = selected.Resolve(typecatalog.PackageAuthority, predicate.Key())
	require.NoError(t, err)
	require.True(t, found)
}

func TestRuntimeOnlyDateRangeSurvivesWithoutAuthorizingRows(t *testing.T) {
	catalog, err := predicatecatalog.New(
		predicatecatalog.Package{Path: "github.com/viant/datly/runtime/predicate", Types: []reflect.Type{reflect.TypeFor[rangefilter.DateRange](), reflect.TypeFor[rangefilter.DateRangeInput]()}, AuthorizationTypes: []string{}},
		predicatecatalog.Package{Path: "example.com/unselected", Types: []reflect.Type{reflect.TypeFor[configuredPredicate]()}},
	)
	require.NoError(t, err)
	source, err := catalog.RuntimeTypes()
	require.NoError(t, err)
	policy := &settings.Namespace{Security: &settings.Security{RequireSecurityPredicate: true, PredicatePackages: []string{"example.com/authorized"}}}
	selected, err := predicatepolicy.Types(source, policy, catalog)
	require.NoError(t, err)
	_, found, err := selected.Resolve(typecatalog.PackageAuthority, "github.com/viant/datly/runtime/predicate.DateRange")
	require.NoError(t, err)
	require.True(t, found, "explicit runtime-only filter must remain compilable")
	_, found, err = selected.Resolve(typecatalog.PackageAuthority, "example.com/unselected.configuredPredicate")
	require.NoError(t, err)
	require.False(t, found, "unselected authorization handlers must remain excluded")
	require.False(t, catalog.Contains("github.com/viant/datly/runtime/predicate", "DateRange"))
	required := true
	spoofed := &spec.Component{Parameters: []*spec.Parameter{{Required: &required, Predicates: []*spec.Predicate{{Auth: true, Name: "handler", Args: []string{"github.com/viant/datly/runtime/predicate.DateRange"}}}}}}
	require.Error(t, predicatepolicy.ValidateDeclarations(spoofed, policy), "date filter cannot satisfy the selected authorization package requirement")
}
