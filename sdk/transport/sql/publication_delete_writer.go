package sqltransport

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/viant/bindly/locator"
	"github.com/viant/bindly/resource"
	stored "github.com/viant/datly-studio/studio/report_publications/store_delete"
	"github.com/viant/datly/bootstrap"
	dexec "github.com/viant/datly/exec"
	druntime "github.com/viant/datly/runtime"
	writerhandler "github.com/viant/datly/runtime/handler/writer"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	dsql "github.com/viant/datly/sql"
	"github.com/viant/datly/sql/dml"
	viewprovider "github.com/viant/datly/sql/reader/provider"
	dtag "github.com/viant/datly/tag"
)

func (t *Transport) writePublicationDelete(ctx context.Context, tx *sql.Tx, row *stored.StoredPublication) error {
	resources := resource.New()
	if err := resources.Register(stored.PublicationDatlyResourceNamespace, stored.PublicationDatlyResources); err != nil {
		return err
	}
	connector := &dsql.SQLComponent{DB: t.DB, Tx: tx}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return err
	}
	holder := reflect.TypeOf(stored.PublicationComponent{})
	contract, ok := holder.FieldByName("Contract")
	if !ok {
		return fmt.Errorf("publication delete writer has no component contract")
	}
	metadata, present, err := dtag.ParseComponent(contract.Tag)
	if err != nil {
		return err
	}
	if !present {
		return fmt.Errorf("publication delete writer has no component metadata")
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: contract.Name,
		PackageName: "store_delete", PackagePath: holder.PkgPath(), Tag: metadata,
		InputType: "Input", OutputType: "Output"}).Resolve(reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}))
	if err != nil {
		return err
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: reflect.TypeOf(stored.Input{}), OutputType: reflect.TypeOf(stored.Output{}), Resources: resources})
	if err != nil {
		return err
	}
	var providers []locator.Provider
	if len(artifact.ViewDependencies) > 0 {
		views, err := viewprovider.New(viewprovider.Config{Dependencies: artifact.ViewDependencies, Input: artifact.Input, SQL: connector})
		if err != nil {
			return err
		}
		providers = append(providers, views)
	}
	handler, err := writerhandler.New(component, reflect.TypeOf(stored.Input{}), reflect.TypeOf(stored.Output{}), "patch")
	if err != nil {
		return err
	}
	registration := &registry.RegisteredComponent{Component: artifact.Component, Input: artifact.Input,
		Output: artifact.Output, OutputType: reflect.TypeOf(stored.Output{}), Handler: handler,
		Providers: providers, DataSource: dml.Source{DB: t.DB, Tx: tx}}
	registration.Capabilities.Connector = connector
	runtime, err := druntime.NewRuntime([]*registry.RegisteredComponent{registration}, druntime.WithResources(resources))
	if err != nil {
		return err
	}
	defer runtime.Shutdown(context.Background())
	target := dexec.ComponentTarget{Component: component.Key}
	if len(component.Routes) > 0 && component.Routes[0] != nil {
		target.Route = spec.RouteRef{Method: component.Routes[0].Method, Path: component.Routes[0].Path}
	}
	input := &stored.Input{}
	input.SetPublications([]*stored.StoredPublication{row})
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target, Input: input})
	if err != nil {
		return err
	}
	if _, ok := value.(*stored.Output); !ok {
		return fmt.Errorf("publication delete writer returned %T", value)
	}
	return nil
}
