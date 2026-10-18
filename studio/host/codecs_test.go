package host_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/viant/datly-studio/studio/codeccatalog"
	"github.com/viant/datly-studio/studio/host"
	"github.com/viant/datly/typecatalog"
	xcodec "github.com/viant/xdatly/codec"
)

type hostQueryCodec struct{}

func (*hostQueryCodec) ResultType(reflect.Type) (reflect.Type, error) {
	return reflect.TypeFor[string](), nil
}
func (*hostQueryCodec) Value(context.Context, any, ...xcodec.Option) (any, error) { return "test", nil }

func TestCodecExtensionsRemainSeparateAndFrozen(t *testing.T) {
	typ := reflect.TypeFor[hostQueryCodec]()
	config := host.AuthoringConfig{Config: host.Config{CodecPackages: []codeccatalog.Package{{Path: typ.PkgPath(), Types: []reflect.Type{typ}}}}}
	snapshot, err := host.NewAuthoring(config)
	if err != nil {
		t.Fatal(err)
	}
	config.Config.CodecPackages[0].Path = "example.com/changed"
	config.Config.CodecPackages[0].Types[0] = nil
	types, err := snapshot.RuntimeTypes()
	if err != nil {
		t.Fatal(err)
	}
	actual, found, err := types.ResolveRuntimeType(typecatalog.PackageAuthority, typ.PkgPath()+"."+typ.Name())
	if err != nil || !found || actual != typ {
		t.Fatalf("codec=%v found=%v err=%v", actual, found, err)
	}
	if snapshot.PredicateCatalog().Contains(typ.PkgPath(), typ.Name()) {
		t.Fatal("codec exposed as an authorization predicate")
	}
}
