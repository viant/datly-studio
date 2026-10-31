// Package writercomponent binds private transcribed writers to the host connector.
package writercomponent

import (
	"database/sql"
	"fmt"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly/bootstrap"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/runtime/handler/writer"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/sql/dml"
	"github.com/viant/datly/typecatalog"
	"reflect"
)

// Compile retains native binding, Current reads, validation and transaction ownership.
func Compile(holder, inputType, outputType reflect.Type, resources *resource.Store, connector *dsql.SQLComponent, db *sql.DB, types *typecatalog.Catalog) (*registry.RegisteredComponent, exec.ComponentTarget, error) {
	sources, err := bootstrap.ReflectPackages([]string{holder.PkgPath()})
	if err != nil {
		return nil, exec.ComponentTarget{}, err
	}
	for _, source := range sources.Components {
		if source.HolderType != holder.Name() {
			continue
		}
		component, err := source.Resolve(inputType, outputType)
		if err != nil {
			return nil, exec.ComponentTarget{}, err
		}
		handler, err := writer.NewWithIdentifierProducts(component, inputType, outputType, component.Settings.Mutation, connector.IdentifierProduct)
		if err != nil {
			return nil, exec.ComponentTarget{}, err
		}
		artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: inputType, OutputType: outputType, Resources: resources, Types: types, Handler: handler, HandlerOwnedOutput: true, IdentifierProducts: connector.IdentifierProduct})
		if err != nil {
			return nil, exec.ComponentTarget{}, err
		}
		registration := registry.RegisteredComponent{DataSource: dml.Source{DB: db}}
		registration.Capabilities.Connector = connector
		provider, err := artifact.NewViewProvider(bootstrap.ViewRuntimeConfig{SQL: connector})
		if err != nil {
			return nil, exec.ComponentTarget{}, err
		}
		if provider != nil {
			registration.Providers = append(registration.Providers, provider)
		}
		compiled, err := artifact.Registration(registration)
		if err != nil {
			return nil, exec.ComponentTarget{}, err
		}
		target := exec.ComponentTarget{Component: component.Key}
		if len(component.Routes) > 0 && component.Routes[0] != nil {
			target.Route = spec.RouteRef{Method: component.Routes[0].Method, Path: component.Routes[0].Path}
		}
		return compiled, target, nil
	}
	return nil, exec.ComponentTarget{}, fmt.Errorf("writer component %s is not linked", holder)
}
