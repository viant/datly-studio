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
	"github.com/viant/datly-studio/studio/host"
	acl "github.com/viant/datly-studio/studio/report_acl/reader"
	"github.com/viant/datly/bootstrap"
	"github.com/viant/datly/gateway/openapi"
	"github.com/viant/datly/gateway/openapi/openapi3"
	runtimeauth "github.com/viant/datly/runtime/auth"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/tag"
	"github.com/viant/scy"
	"github.com/viant/scy/auth/jwt/verifier"
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
	holder := reflect.TypeFor[acl.AclComponent]()
	field, ok := holder.FieldByName("Contract")
	if !ok {
		return fmt.Errorf("ACL component contract is missing")
	}
	metadata, present, err := tag.ParseComponent(field.Tag)
	if err != nil || !present {
		return fmt.Errorf("parse ACL component metadata: %w", err)
	}
	component, err := (&bootstrap.RouteSource{HolderType: holder.Name(), FieldName: field.Name,
		PackageName: "reader", PackagePath: holder.PkgPath(), Tag: metadata}).Resolve(
		reflect.TypeFor[acl.Input](), reflect.TypeFor[acl.Output]())
	if err != nil {
		return err
	}
	resources := resource.New()
	if err = resources.Register(acl.AclDatlyResourceNamespace, acl.AclDatlyResources); err != nil {
		return err
	}
	predicates, err := (host.Config{}).PredicateCatalog()
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
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component,
		InputType: reflect.TypeFor[acl.Input](), OutputType: reflect.TypeFor[acl.Output](),
		Resources: resources, Types: types, CodecFactory: codec})
	if err != nil {
		return err
	}
	registered := &registry.RegisteredComponent{Component: artifact.Component, Input: artifact.Input,
		Output: artifact.Output, OutputType: reflect.TypeFor[acl.Output]()}
	document, err := (openapi.Generator{}).Generate(ctx, openapi.Request{
		Info:       openapi3.Info{Title: "Datly Studio SDK", Version: "1.0.0"},
		Components: []*registry.RegisteredComponent{registered},
		Routes:     []spec.RouteRef{{Method: "POST", Path: "/v1/studio/sdk/acl.list"}},
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
