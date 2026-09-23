package predicatecatalog

import (
	"context"
	"github.com/viant/datly/typecatalog"
	"github.com/viant/xdatly/predicate"
	"reflect"
	"testing"
)

type testPredicate struct{}

func (*testPredicate) Compute(context.Context, any) (*predicate.Criteria, error) {
	return &predicate.Criteria{Expression: "tenant_id = ?", Placeholders: []any{42}}, nil
}

func TestRuntimeCatalogRetainsExecutableCompute(t *testing.T) {
	catalog, err := New(Package{Path: "example.com/authorization", Types: []reflect.Type{reflect.TypeOf((*testPredicate)(nil))}})
	if err != nil {
		t.Fatal(err)
	}
	types, err := catalog.RuntimeTypes()
	if err != nil {
		t.Fatal(err)
	}
	typ, found, err := types.ResolveRuntimeType(typecatalog.PackageAuthority, "example.com/authorization.testPredicate")
	if err != nil || !found || typ != reflect.TypeOf(testPredicate{}) {
		t.Fatalf("runtime type=%v found=%v err=%v", typ, found, err)
	}
	handler := reflect.New(typ).Interface().(interface {
		Compute(context.Context, any) (*predicate.Criteria, error)
	})
	criteria, err := handler.Compute(context.Background(), nil)
	if err != nil || criteria.Expression != "tenant_id = ?" || criteria.Placeholders[0] != 42 {
		t.Fatalf("criteria=%+v error=%v", criteria, err)
	}
	second, err := catalog.RuntimeTypes()
	if err != nil {
		t.Fatal(err)
	}
	if second == types {
		t.Fatal("mutable runtime catalogs are shared")
	}
}

func TestCatalogLinksExplicitPredicatePackages(t *testing.T) {
	catalog, err := New(Package{Alias: "authz", Path: "example.com/project/authorization", Types: []reflect.Type{reflect.TypeOf(testPredicate{})}})
	if err != nil {
		t.Fatal(err)
	}
	if !catalog.Contains("example.com/project/authorization", "testPredicate") || catalog.Contains("example.com/project/authorization", "Missing") {
		t.Fatalf("catalog=%+v", catalog.Descriptors())
	}
}
