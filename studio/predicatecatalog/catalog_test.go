package predicatecatalog

import (
	"reflect"
	"testing"
)

type testPredicate struct{}

func TestCatalogLinksExplicitPredicatePackages(t *testing.T) {
	catalog, err := New(Package{Alias: "authz", Path: "example.com/project/authorization", Types: []reflect.Type{reflect.TypeOf(testPredicate{})}})
	if err != nil {
		t.Fatal(err)
	}
	if !catalog.Contains("example.com/project/authorization", "testPredicate") || catalog.Contains("example.com/project/authorization", "Missing") {
		t.Fatalf("catalog=%+v", catalog.Descriptors())
	}
}
