// studio-openapi exports the public Datly SDK contract for client generation.
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/viant/bindly/resource"
	authreader "github.com/viant/datly-studio/studio/auth/reader"
	connectorget "github.com/viant/datly-studio/studio/connectors/get"
	connectors "github.com/viant/datly-studio/studio/connectors/reader"
	"github.com/viant/datly-studio/studio/host"
	namespaceget "github.com/viant/datly-studio/studio/namespaces/get"
	namespaces "github.com/viant/datly-studio/studio/namespaces/reader"
	"github.com/viant/datly-studio/studio/predicatecatalog"
	acl "github.com/viant/datly-studio/studio/report_acl/reader"
	catalogpredicate "github.com/viant/datly-studio/studio/reports/catalogpredicate"
	reportget "github.com/viant/datly-studio/studio/reports/get"
	reports "github.com/viant/datly-studio/studio/reports/reader"
	"github.com/viant/datly/bootstrap"
	"github.com/viant/datly/gateway/openapi"
	"github.com/viant/datly/gateway/openapi/openapi3"
	runtimeauth "github.com/viant/datly/runtime/auth"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/tag"
	"github.com/viant/datly/typecatalog"
	"github.com/viant/scy"
	"github.com/viant/scy/auth/jwt/verifier"
	xcodec "github.com/viant/xdatly/codec"
)

func main() {
	output := flag.String("out", "sdk/openapi/studio.json", "generated Studio SDK OpenAPI document")
	flag.Parse()
	if err := run(context.Background(), *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, output string) error {
	resources := resource.New()
	if err := resources.Register(acl.AclDatlyResourceNamespace, acl.AclDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(connectors.ConnectorDatlyResourceNamespace, connectors.ConnectorDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(connectorget.ConnectorDatlyResourceNamespace, connectorget.ConnectorDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(namespaces.NamespaceDatlyResourceNamespace, namespaces.NamespaceDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(namespaceget.NamespaceDatlyResourceNamespace, namespaceget.NamespaceDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(authreader.ContextDatlyResourceNamespace, authreader.ContextDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(reports.ReportDatlyResourceNamespace, reports.ReportDatlyResources); err != nil {
		return err
	}
	if err := resources.Register(reportget.ReportDatlyResourceNamespace, reportget.ReportDatlyResources); err != nil {
		return err
	}
	predicates, err := (host.Config{PredicatePackages: []predicatecatalog.Package{{
		Alias: "catalogpredicate", Path: "github.com/viant/datly-studio/studio/reports/catalogpredicate",
		Types: []reflect.Type{reflect.TypeFor[catalogpredicate.ReportCatalogRead]()},
	}}}).PredicateCatalog()
	if err != nil {
		return err
	}
	types, err := predicates.RuntimeTypes()
	if err != nil {
		return err
	}
	// OpenAPI only treats Authorization as bearer security when the compiled
	// codec is a real verifier. A throwaway key keeps generation offline while
	// exercising the same verified-JWT contract as the serving Datly runtime.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return err
	}
	codec, err := runtimeauth.New(ctx, &runtimeauth.Config{JWTValidator: &verifier.Config{RSA: []*scy.Resource{{
		URL: "studio-openapi-ephemeral", Data: pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER}),
	}}}})
	if err != nil {
		return err
	}
	auth, err := compile(reflect.TypeFor[authreader.ContextComponent](), reflect.TypeFor[authreader.Input](),
		reflect.TypeFor[authreader.Output](), resources, types, codec)
	if err != nil {
		return err
	}
	aclList, err := compile(reflect.TypeFor[acl.AclComponent](), reflect.TypeFor[acl.Input](),
		reflect.TypeFor[acl.Output](), resources, types, codec)
	if err != nil {
		return err
	}
	connectorList, err := compile(reflect.TypeFor[connectors.ConnectorComponent](), reflect.TypeFor[connectors.Input](),
		reflect.TypeFor[connectors.Output](), resources, types, codec)
	if err != nil {
		return err
	}
	connectorOne, err := compile(reflect.TypeFor[connectorget.ConnectorComponent](), reflect.TypeFor[connectorget.ConnectorGetInput](),
		reflect.TypeFor[connectorget.ConnectorGetOutput](), resources, types, codec)
	if err != nil {
		return err
	}
	namespaceList, err := compile(reflect.TypeFor[namespaces.NamespaceComponent](), reflect.TypeFor[namespaces.NamespaceQueryInput](),
		reflect.TypeFor[namespaces.NamespaceQueryOutput](), resources, types, codec)
	if err != nil {
		return err
	}
	namespaceOne, err := compile(reflect.TypeFor[namespaceget.NamespaceComponent](), reflect.TypeFor[namespaceget.NamespaceGetInput](),
		reflect.TypeFor[namespaceget.NamespaceGetOutput](), resources, types, codec)
	if err != nil {
		return err
	}
	reportList, err := compile(reflect.TypeFor[reports.ReportComponent](), reflect.TypeFor[reports.Input](),
		reflect.TypeFor[reports.Output](), resources, types, codec)
	if err != nil {
		return err
	}
	reportOne, err := compile(reflect.TypeFor[reportget.ReportComponent](), reflect.TypeFor[reportget.ReportGetInput](),
		reflect.TypeFor[reportget.ReportGetOutput](), resources, types, codec)
	if err != nil {
		return err
	}
	document, err := (openapi.Generator{}).Generate(ctx, openapi.Request{
		Info:       openapi3.Info{Title: "Datly Studio SDK", Version: "1.0.0"},
		Components: []*registry.RegisteredComponent{auth, aclList, connectorList, connectorOne, namespaceList, namespaceOne, reportList, reportOne},
		Routes: []spec.RouteRef{
			{Method: "POST", Path: "/v1/studio/sdk/acl.list"},
			{Method: "POST", Path: "/v1/studio/sdk/connectors.get"},
			{Method: "POST", Path: "/v1/studio/sdk/connectors.list"},
			{Method: "POST", Path: "/v1/studio/sdk/namespaces.list"},
			{Method: "POST", Path: "/v1/studio/sdk/namespaces.get"},
			{Method: "POST", Path: "/v1/studio/sdk/reports.get"},
			{Method: "POST", Path: "/v1/studio/sdk/reports.list"},
		},
	})
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err = os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	return os.WriteFile(output, data, 0o644)
}

func compile(holder, inputType, outputType reflect.Type, resources *resource.Store,
	types *typecatalog.Catalog, codec xcodec.Factory) (*registry.RegisteredComponent, error) {
	field, ok := holder.FieldByName("Contract")
	if !ok {
		return nil, fmt.Errorf("component %s contract is missing", holder)
	}
	metadata, present, err := tag.ParseComponent(field.Tag)
	if err != nil || !present {
		return nil, fmt.Errorf("parse %s metadata: %w", holder, err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name,
		PackageName: "reader", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(inputType, outputType)
	if err != nil {
		return nil, err
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: inputType, OutputType: outputType, Resources: resources, Types: types, CodecFactory: codec})
	if err != nil {
		return nil, err
	}
	return &registry.RegisteredComponent{Component: artifact.Component, Input: artifact.Input,
		Output: artifact.Output, OutputType: outputType}, nil
}
