package predicatepolicy_test

import (
	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/examples/predicates"
	"github.com/viant/datly-studio/internal/predicatepolicy"
	"github.com/viant/datly-studio/studio/settings"
	"github.com/viant/datly/typecatalog"
	x "github.com/viant/x"
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
