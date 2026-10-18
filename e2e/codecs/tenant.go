// Package codecs provides synthetic, host-linked codecs for the Endly suite.
package codecs

import (
	"context"
	"fmt"
	"github.com/viant/datly-studio/studio/codeccatalog"
	"github.com/viant/datly-studio/studio/host"
	xcodec "github.com/viant/xdatly/codec"
	"reflect"
	"strings"
)

// Tenant normalizes a tenant selector and rejects invalid values before SQL runs.
type Tenant struct{}

func (*Tenant) New(*xcodec.Config, ...xcodec.Option) (xcodec.Instance, error) { return &Tenant{}, nil }
func (*Tenant) ResultType(reflect.Type) (reflect.Type, error)                 { return reflect.TypeFor[string](), nil }
func (*Tenant) Value(_ context.Context, raw any, _ ...xcodec.Option) (any, error) {
	value, ok := raw.(string)
	if !ok {
		return nil, fmt.Errorf("tenant must be a string")
	}
	value = strings.ToLower(strings.TrimSpace(value))
	if value != "alpha" && value != "beta" {
		return nil, fmt.Errorf("unknown synthetic tenant")
	}
	return value, nil
}
func init() {
	if err := host.RegisterCodecPackage(codeccatalog.Package{Path: reflect.TypeFor[Tenant]().PkgPath(), Types: []reflect.Type{reflect.TypeFor[Tenant]()}}); err != nil {
		panic(err)
	}
}
