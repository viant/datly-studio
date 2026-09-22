package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly/bootstrap"
	standaloneconfig "github.com/viant/datly/standalone/config"
	"github.com/viant/datly/transcribe"
)

func TestRuntimeReflectionTypesCompileConnectorReaderDQL(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	configuration, err := (standaloneconfig.Loader{}).Load(context.Background(), filepath.Join(root, "datly.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	reflected, err := bootstrap.ReflectPackages(configuration.GoBootstrap.Packages)
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, "dql", "studio", "connectors", "reader")
	payload, err := os.ReadFile(filepath.Join(directory, "connector.dql"))
	if err != nil {
		t.Fatal(err)
	}
	resources, err := resource.New().WithDefault(os.DirFS(directory))
	if err != nil {
		t.Fatal(err)
	}
	_, err = transcribe.NewCompiler().Compile(context.Background(), &transcribe.Source{
		Scope: "github.com/viant/datly-studio/studio/connectors/reader", Name: "connector", Text: string(payload), Connector: "studio", Resources: resources, Types: reflected.Types,
	})
	if err != nil {
		t.Fatalf("compile connector DQL with runtime reflection types: %v", err)
	}
}

func TestStudioDQLPackageUsesProjectDQLRoot(t *testing.T) {
	configured := []string{"github.com/viant/datly-studio/studio/connectors/reader"}
	if actual := studioDQLPackage("github.com/viant/datly-studio/studio/connectors/reader", configured); actual != "github.com/viant/datly-studio/dql/studio/connectors/reader" {
		t.Fatalf("DQL package = %q", actual)
	}
	if actual := studioDQLPackage("example.com/external/reader", configured); actual != "example.com/external/reader" {
		t.Fatalf("external package = %q", actual)
	}
}
