// Package accesscatalog reads resource metadata through an internal Datly
// component. It has no policy-writing or business-authorization capability.
package accesscatalog

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/readercomponent"
	catalog "github.com/viant/datly-studio/studio/resource_policy/catalog"
	"github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	"reflect"
)

type Store struct {
	DB      *sql.DB
	Invoker exec.ComponentInvoker
}

// Catalog returns candidates. Callers must authorize visibility before
// returning their names, identifiers, counts or pagination results.
func (s *Store) Catalog(ctx context.Context, limit, offset int) ([]*catalog.Entry, error) {
	input := &catalog.Input{Limit: limit, Offset: offset}
	invoker := s.Invoker
	target := exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[catalog.Component]().PkgPath(), Name: "catalog"}, Route: spec.RouteRef{Method: "GET", Path: "/_studio/resource-policy-store/catalog"}}
	if invoker == nil {
		resources := resource.New()
		if err := resources.Register(catalog.Namespace, catalog.Resources); err != nil {
			return nil, err
		}
		sqlComponent := &dsql.SQLComponent{DB: s.DB}
		if err := sqlComponent.RegisterConnector("studio", s.DB); err != nil {
			return nil, err
		}
		registration, resolved, err := readercomponent.Compile(reflect.TypeFor[catalog.Component](), "catalog", reflect.TypeFor[catalog.Input](), reflect.TypeFor[catalog.Output](), resources, sqlComponent)
		if err != nil {
			return nil, err
		}
		runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
		if err != nil {
			return nil, err
		}
		invoker = runtime
		target = resolved
	}
	value, err := invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*catalog.Output)
	if !ok || output == nil {
		return nil, fmt.Errorf("resource catalog returned %T", value)
	}
	return output.Items, nil
}
