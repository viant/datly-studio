package predicateprojection_test

import (
	"context"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/internal/predicateprojection"
	stored "github.com/viant/datly-studio/studio/authorization_predicates/store_read"
	"github.com/viant/datly/bootstrap"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
	"reflect"
	"testing"
)

func TestReadEmptyPredicateCatalogThroughNativeRuntime(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "predicate_read", "studio")
	typ := reflect.TypeFor[stored.AuthorizationPredicateComponent]()
	reflected, err := bootstrap.ReflectSelectedPackages([]string{typ.PkgPath()}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(reflected.Components) != 1 {
		t.Fatalf("expected one linked predicate reader; got %d", len(reflected.Components))
	}
	component, err := reflected.Components[0].Resolve(reflect.TypeFor[stored.Input](), reflect.TypeFor[stored.Output]())
	if err != nil {
		t.Fatal(err)
	}
	if component.SelectorCompanion == nil {
		t.Fatal("reader must retain its side-input query selectors")
	}
	for _, name := range []string{"OrderBy", "Limit", "Offset"} {
		if _, found := reflect.TypeFor[stored.Input]().FieldByName(name); found {
			t.Fatalf("query selector %s leaked into ordinary reader input", name)
		}
	}
	resources := resource.New()
	if err = resources.Register(stored.AuthorizationPredicateDatlyResourceNamespace, stored.AuthorizationPredicateDatlyResources); err != nil {
		t.Fatal(err)
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeFor[stored.Input](), OutputType: reflect.TypeFor[stored.Output](), Resources: resources})
	if err != nil {
		t.Fatal(err)
	}
	sqlComponent := &dsql.SQLComponent{DB: db}
	if err = sqlComponent.RegisterConnector("studio", db); err != nil {
		t.Fatal(err)
	}
	execution, err := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: sqlComponent})
	if err != nil {
		t.Fatal(err)
	}
	entry, err := artifact.Registration(registry.RegisteredComponent{Reader: execution})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{entry}, druntime.WithResources(resources))
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Shutdown(ctx)
	rows, err := predicateprojection.Read(ctx, runtime, "", "", "", 26, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows=%d", len(rows))
	}
}
