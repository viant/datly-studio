package codeccatalog_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/datly-studio/studio/codeccatalog"
	"github.com/viant/datly/bootstrap"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/typecatalog"
	xcodec "github.com/viant/xdatly/codec"
)

type QueryOrder struct{}

func (*QueryOrder) New(*xcodec.Config, ...xcodec.Option) (xcodec.Instance, error) {
	return &QueryOrder{}, nil
}
func (*QueryOrder) ResultType(reflect.Type) (reflect.Type, error) {
	return reflect.TypeFor[string](), nil
}
func (*QueryOrder) Value(_ context.Context, value any, _ ...xcodec.Option) (any, error) {
	return strings.Join(value.([]string), ","), nil
}

type NotCodec struct{}

func TestCodecPackageDiscoveryAndCompilation(t *testing.T) {
	typ := reflect.TypeFor[QueryOrder]()
	catalog := typecatalog.NewCatalog()
	if err := codeccatalog.Link(catalog, codeccatalog.Package{Path: typ.PkgPath()}); err != nil {
		t.Fatal(err)
	}
	actual, found, err := catalog.ResolveRuntimeType(typecatalog.PackageAuthority, typ.PkgPath()+".QueryOrder")
	if err != nil || !found || actual != typ {
		t.Fatalf("codec=%v found=%v err=%v", actual, found, err)
	}
	if _, found, err := catalog.ResolveRuntimeType(typecatalog.PackageAuthority, typ.PkgPath()+".NotCodec"); err != nil || found {
		t.Fatalf("non-codec discovered: %v %v", found, err)
	}
	source := `#setting($_ = $route('/sample', 'GET'))
#package('example.com/readers/sample')
#set($_ = $OrderBy<[]string,string>(form/sort).WithCodec('` + typ.PkgPath() + `.QueryOrder').QuerySelector(sample).Optional())
#set($_ = $Data<?>(output/view))
SELECT sample.*, selector_order_by(sample, true) FROM (SELECT 1 AS id) sample`
	contract, err := transcribe.NewCompiler().RuntimeContractsInModule(context.Background(), "example.com/readers", &transcribe.Source{Scope: "example.com/readers/sample", Name: "sample", Text: source, Types: catalog})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: contract.Component, InputType: contract.InputType, OutputType: contract.OutputType, Types: contract.Types, Resources: contract.Resources}); err != nil {
		t.Fatal(err)
	}
}
func TestCodecPackageRejectsUnlinkedAndInvalidTypes(t *testing.T) {
	typ := reflect.TypeFor[QueryOrder]()
	for name, pkg := range map[string]codeccatalog.Package{
		"unlinked":       {Path: "example.com/unlinked"},
		"wrong package":  {Path: "example.com/wrong", Types: []reflect.Type{typ}},
		"non-codec":      {Path: typ.PkgPath(), Types: []reflect.Type{reflect.TypeFor[NotCodec]()}},
		"nil type":       {Path: typ.PkgPath(), Types: []reflect.Type{nil}},
		"duplicate type": {Path: typ.PkgPath(), Types: []reflect.Type{typ, typ}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := codeccatalog.Link(typecatalog.NewCatalog(), pkg); err == nil {
				t.Fatal("invalid codec package accepted")
			}
		})
	}
}
