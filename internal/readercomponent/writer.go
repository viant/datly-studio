package readercomponent

import (
	"fmt"
	"github.com/viant/bindly/locator"
	"github.com/viant/bindly/resource"
	"github.com/viant/datly/bootstrap"
	dexec "github.com/viant/datly/exec"
	writerhandler "github.com/viant/datly/runtime/handler/writer"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/sql/dml"
	viewprovider "github.com/viant/datly/sql/reader/provider"
	dtag "github.com/viant/datly/tag"
	"reflect"
)

// CompileWriter uses the same native generated writer assembly as warmup persistence.
func CompileWriter(holder reflect.Type, packageName string, inputType, outputType reflect.Type, resources *resource.Store, connector *dsql.SQLComponent) (*registry.RegisteredComponent, dexec.ComponentTarget, error) {
	contract, ok := holder.FieldByName("Contract")
	if !ok {
		return nil, dexec.ComponentTarget{}, fmt.Errorf("native store writer has no component contract")
	}
	metadata, present, err := dtag.ParseComponent(contract.Tag)
	if err != nil {
		return nil, dexec.ComponentTarget{}, err
	}
	if !present {
		return nil, dexec.ComponentTarget{}, fmt.Errorf("native store writer has no component metadata")
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: contract.Name,
		PackageName: packageName, PackagePath: holder.PkgPath(), Tag: metadata,
		InputType: "Input", OutputType: "Output"}).Resolve(inputType, outputType)
	if err != nil {
		return nil, dexec.ComponentTarget{}, err
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: inputType, OutputType: outputType, Resources: resources, IdentifierProducts: connector.IdentifierProduct})
	if err != nil {
		return nil, dexec.ComponentTarget{}, err
	}
	views, err := viewprovider.New(viewprovider.Config{Dependencies: artifact.ViewDependencies, Input: artifact.Input, SQL: connector})
	if err != nil {
		return nil, dexec.ComponentTarget{}, err
	}
	handler, err := writerhandler.New(component, inputType, outputType, "patch")
	if err != nil {
		return nil, dexec.ComponentTarget{}, err
	}
	registration := &registry.RegisteredComponent{Component: artifact.Component, Input: artifact.Input,
		Output: artifact.Output, OutputType: outputType, Handler: handler,
		Providers: []locator.Provider{views}, DataSource: dml.Source{DB: connector.DB}}
	registration.Capabilities.Connector = connector
	target := dexec.ComponentTarget{Component: component.Key}
	if len(component.Routes) > 0 && component.Routes[0] != nil {
		target.Route = spec.RouteRef{Method: component.Routes[0].Method, Path: component.Routes[0].Path}
	}
	return registration, target, nil
}
