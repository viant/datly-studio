package sqltransport

import (
	"context"
	"fmt"
	"reflect"

	"github.com/viant/bindly/locator"
	"github.com/viant/bindly/resource"
	storedwriter "github.com/viant/datly-studio/studio/authorization_predicates/store_write"
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

// writeAuthorizationPredicate invokes the generated server-owned Datly writer.
// Invoke authorizes the SDK operation before this method is reached.
func (t *Transport) writeAuthorizationPredicate(ctx context.Context, operation string, row *storedwriter.StoredAuthorizationPredicate) error {
	resources := resource.New()
	if err := resources.Register(storedwriter.AuthorizationPredicateDatlyResourceNamespace, storedwriter.AuthorizationPredicateDatlyResources); err != nil {
		return err
	}
	connector := &dsql.SQLComponent{DB: t.DB}
	if err := connector.RegisterConnector("studio", t.DB); err != nil {
		return err
	}
	holder := reflect.TypeOf(storedwriter.AuthorizationPredicateComponent{})
	contract, ok := holder.FieldByName("Contract")
	if !ok {
		return fmt.Errorf("authorization predicate store writer has no component contract")
	}
	metadata, present, err := dtag.ParseComponent(contract.Tag)
	if err != nil {
		return fmt.Errorf("authorization predicate store writer metadata: %w", err)
	}
	if !present {
		return fmt.Errorf("authorization predicate store writer has no component metadata")
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: contract.Name,
		PackageName: "store_write", PackagePath: holder.PkgPath(), Tag: metadata,
		InputType: "Input", OutputType: "Output"}).Resolve(reflect.TypeOf(storedwriter.Input{}), reflect.TypeOf(storedwriter.Output{}))
	if err != nil {
		return err
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: reflect.TypeOf(storedwriter.Input{}), OutputType: reflect.TypeOf(storedwriter.Output{}), Resources: resources})
	if err != nil {
		return err
	}
	views, err := viewprovider.New(viewprovider.Config{Dependencies: artifact.ViewDependencies, Input: artifact.Input, SQL: connector})
	if err != nil {
		return err
	}
	handler, err := writerhandler.New(component, reflect.TypeOf(storedwriter.Input{}), reflect.TypeOf(storedwriter.Output{}), operation)
	if err != nil {
		return err
	}
	registration := &registry.RegisteredComponent{Component: artifact.Component, Input: artifact.Input,
		Output: artifact.Output, OutputType: reflect.TypeOf(storedwriter.Output{}), Handler: handler,
		Providers: []locator.Provider{views}, DataSource: dml.Source{DB: t.DB}}
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
	value, err := runtime.InvokeComponent(ctx, dexec.ComponentRequest{Target: target,
		Input: &storedwriter.Input{AuthorizationPredicates: []*storedwriter.StoredAuthorizationPredicate{row},
			Has: &storedwriter.InputHas{AuthorizationPredicates: true}}})
	if err != nil {
		return err
	}
	if _, ok := value.(*storedwriter.Output); !ok {
		return fmt.Errorf("authorization predicate writer returned %T", value)
	}
	return nil
}
