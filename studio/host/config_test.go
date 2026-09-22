package host_test

import (
	"reflect"
	"testing"

	"github.com/viant/datly-studio/studio/host"
	"github.com/viant/datly-studio/studio/predicatecatalog"
)

type tenantPredicate struct{}

func TestConfigPredicateCatalogIncludesHostPackages(t *testing.T) {
	catalog, err := (host.Config{PredicatePackages: []predicatecatalog.Package{{Alias: "tenant", Path: "example.com/acme/iam", Types: []reflect.Type{reflect.TypeFor[tenantPredicate]()}}}}).PredicateCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if !catalog.Contains("example.com/acme/iam", "tenantPredicate") {
		t.Fatal("host predicate type was not registered")
	}
}
