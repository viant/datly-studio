package datatest

import (
	"database/sql"
	"reflect"
	"testing"

	"github.com/viant/bindly/resource"
	authreader "github.com/viant/datly-studio/studio/auth/reader"
	"github.com/viant/datly/bootstrap"
	"github.com/viant/datly/runtime/registry"
	dsql "github.com/viant/datly/sql"
	dtag "github.com/viant/datly/tag"
	xcodec "github.com/viant/xdatly/codec"
)

func AuthRegistration(t testing.TB, db *sql.DB, factory xcodec.Factory, resources *resource.Store) *registry.RegisteredComponent {
	t.Helper()
	holder := reflect.TypeFor[authreader.ContextComponent]()
	field, ok := holder.FieldByName("Contract")
	if !ok {
		t.Fatal("missing auth context contract")
	}
	metadata, present, err := dtag.ParseComponent(field.Tag)
	if err != nil || !present {
		t.Fatal(err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name, PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(reflect.TypeFor[authreader.Input](), reflect.TypeFor[authreader.Output]())
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := resources.Lookup(authreader.ContextDatlyResourceNamespace); !exists {
		if err = resources.Register(authreader.ContextDatlyResourceNamespace, authreader.ContextDatlyResources); err != nil {
			t.Fatal(err)
		}
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeFor[authreader.Input](), OutputType: reflect.TypeFor[authreader.Output](), Resources: resources, Types: StudioAuthorizationTypes(t), CodecFactory: factory})
	if err != nil {
		t.Fatal(err)
	}
	execution, err := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: &dsql.SQLComponent{DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	registered, err := artifact.Registration(registry.RegisteredComponent{Reader: execution})
	if err != nil {
		t.Fatal(err)
	}
	return registered
}
