// Package readercomponent compiles generated Datly v1 readers for private,
// in-process use by Studio services. It never installs HTTP/MCP routes.
package readercomponent

import (
	"fmt"
	"reflect"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly/bootstrap"
	dexec "github.com/viant/datly/exec"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	dtag "github.com/viant/datly/tag"
	"github.com/viant/datly/typecatalog"
)

// Compile returns a Datly component registration and its in-process target.
// The route in the generated contract is dispatch metadata, not an exposed API.
func Compile(holder reflect.Type, packageName string, inputType, outputType reflect.Type, resources *resource.Store, connector *dsql.SQLComponent, catalogs ...*typecatalog.Catalog) (*registry.RegisteredComponent, dexec.ComponentTarget, error) {
	contract, ok := holder.FieldByName("Contract")
	if !ok {
		return nil, dexec.ComponentTarget{}, fmt.Errorf("server reader %s has no generated contract", packageName)
	}
	metadata, present, err := dtag.ParseComponent(contract.Tag)
	if err != nil {
		return nil, dexec.ComponentTarget{}, err
	}
	if !present {
		return nil, dexec.ComponentTarget{}, fmt.Errorf("server reader %s has no component metadata", packageName)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: contract.Name,
		PackageName: packageName, PackagePath: holder.PkgPath(), Tag: metadata,
		InputType: inputType.Name(), OutputType: outputType.Name()}).Resolve(inputType, outputType)
	if err != nil {
		return nil, dexec.ComponentTarget{}, err
	}
	var types *typecatalog.Catalog
	if len(catalogs) > 0 {
		types = catalogs[0]
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: inputType, OutputType: outputType, Resources: resources, Types: types})
	if err != nil {
		return nil, dexec.ComponentTarget{}, err
	}
	execution, err := artifact.ReaderCompilation().NewExecution(bootstrap.ReaderRuntimeConfig{SQL: connector})
	if err != nil {
		return nil, dexec.ComponentTarget{}, err
	}
	registration, err := artifact.Registration(registry.RegisteredComponent{Reader: execution})
	if err != nil {
		return nil, dexec.ComponentTarget{}, err
	}
	target := dexec.ComponentTarget{Component: component.Key}
	if len(component.Routes) > 0 && component.Routes[0] != nil {
		target.Route = spec.RouteRef{Method: component.Routes[0].Method, Path: component.Routes[0].Path}
	}
	return registration, target, nil
}
