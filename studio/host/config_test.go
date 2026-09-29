package host_test

import (
	"reflect"
	"testing"

	"github.com/viant/datly-studio/runtime/accesscontext"
	"github.com/viant/datly-studio/studio/host"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	"github.com/viant/datly-studio/studio/predicatecatalog/testdata/extension"
	"github.com/viant/datly/typecatalog"
)

type tenantPredicate struct{}

func TestRuntimeTypesResolveAuthorizationDependenciesAndHostPredicates(t *testing.T) {
	types, err := (host.Config{PredicatePackages: []predicatecatalog.Package{{Alias: "tenant", Path: "example.com/acme/iam", Types: []reflect.Type{reflect.TypeFor[tenantPredicate]()}}}}).RuntimeTypes()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{accesscontext.ImportPath + ".Output", accesscontext.ImportPath + ".Scope", "example.com/acme/iam.tenantPredicate"} {
		if _, found, err := types.ResolveRuntimeType(typecatalog.PackageAuthority, name); err != nil || !found {
			t.Fatalf("runtime dependency %s unavailable: %v", name, err)
		}
	}
}

func TestConfigPredicateCatalogIncludesHostPackages(t *testing.T) {
	catalog, err := (host.Config{PredicatePackages: []predicatecatalog.Package{{Alias: "tenant", Path: "example.com/acme/iam", Types: []reflect.Type{reflect.TypeFor[tenantPredicate]()}}}}).PredicateCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if !catalog.Contains("example.com/acme/iam", "tenantPredicate") {
		t.Fatal("host predicate type was not registered")
	}
}

func TestPredicateCatalogUsesTrustedLinkedPackageAllowlist(t *testing.T) {
	_ = extension.LinkedType
	const packagePath = "github.com/viant/datly-studio/studio/predicatecatalog/testdata/extension"
	t.Setenv("STUDIO_PREDICATE_PACKAGES", packagePath)
	catalog, err := (host.Config{}).PredicateCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if !catalog.Contains(packagePath, "ReaderScope") || catalog.Contains(packagePath, "NotAPredicate") {
		t.Fatalf("native linked descriptors=%+v", catalog.Descriptors())
	}
	t.Setenv("STUDIO_PREDICATE_PACKAGES", "example.com/not-linked")
	if _, err = (host.Config{}).PredicateCatalog(); err == nil {
		t.Fatal("unlinked native predicate package was accepted")
	}
}

func TestPredicateCatalogLimitsExternalAuthorizationTypes(t *testing.T) {
	_ = extension.LinkedType
	const packagePath = "github.com/viant/datly-studio/studio/predicatecatalog/testdata/extension"
	t.Setenv("STUDIO_PREDICATE_PACKAGES", packagePath)
	t.Setenv("STUDIO_AUTH_PREDICATE_TYPES", packagePath+"#ReaderScope")
	catalog, err := (host.Config{}).PredicateCatalog()
	if err != nil || !catalog.Contains(packagePath, "ReaderScope") {
		t.Fatalf("filtered catalog=%+v err=%v", catalog, err)
	}
	t.Setenv("STUDIO_AUTH_PREDICATE_TYPES", packagePath+"#NotAPredicate")
	if _, err := (host.Config{}).PredicateCatalog(); err == nil {
		t.Fatal("non-handler authorization type was accepted")
	}
}
