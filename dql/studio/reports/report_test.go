package reports_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	bootstraproutes "github.com/viant/datly/bootstrap/routes"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
)

func TestReportReaderCompilesEmbeddedSQLAndNativeSDKRoute(t *testing.T) {
	compiled := compileReport(t, context.Background(), "reader", nil)
	if compiled.Component == nil || compiled.Component.RootView == nil {
		t.Fatal("report reader component/root view was not compiled")
	}
	if err := (bootstraproutes.Compiler{Component: compiled.Component}).Compile(); err != nil {
		t.Fatal(err)
	}
	routes := map[string]bool{}
	tools := map[string]bool{}
	for _, route := range compiled.Component.Routes {
		routes[route.Method+" "+route.Path] = true
		for _, exposure := range route.MCP {
			if exposure != nil {
				tools[exposure.Name] = true
			}
		}
	}
	if len(routes) != 1 || !routes["POST /v1/studio/sdk/reports.list"] {
		t.Fatalf("report routes = %#v", routes)
	}
	if len(tools) != 1 || !tools["studio.sdk.reports.list"] {
		t.Fatalf("report MCP tools = %#v", tools)
	}
	if source := compiled.Component.RootView.Source; source == nil || len(source.Embeds) != 1 || source.Embeds[0].Path != "sql/read.sql" {
		t.Fatalf("report embedded SQL = %#v", source)
	}
}

func TestReportWriterDiscoversKeysReferencesAndPatchPolicy(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "writer_compile", "studio")
	compiled := compileReport(t, ctx, "writer", column.New(column.Connections{"studio": db}))
	if err := column.ApplyWriterMetadata(compiled.Component); err != nil {
		t.Fatal(err)
	}
	view := compiled.Component.RootView
	if view == nil {
		t.Fatal("report writer root view is missing")
	}
	var primaryKey, deleteMarker, concurrencyToken bool
	for _, field := range view.Columns {
		if field == nil {
			continue
		}
		primaryKey = primaryKey || field.Name == "id" && field.PrimaryKey
		deleteMarker = deleteMarker || field.DeleteMarker
		concurrencyToken = concurrencyToken || field.ConcurrencyToken
	}
	if !primaryKey || !deleteMarker || !concurrencyToken {
		t.Fatalf("report policy: pk=%v delete=%v concurrency=%v", primaryKey, deleteMarker, concurrencyToken)
	}
}

func TestReportTranscribesGoShapesToDeclaredPackages(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "runtime_compile", "studio")
	module := datatest.NewGeneratedModule(t)
	for _, component := range []struct {
		name      string
		operation string
	}{
		{name: "reader", operation: "get"},
		{name: "writer", operation: "patch"},
	} {
		compiled := compileReport(t, ctx, component.name, column.New(column.Connections{"studio": db}))
		generated, err := (transcribe.Generator{Operation: component.operation, EphemeralOwnership: true}).Generate(ctx, transcribe.GenerationRequest{Compiled: compiled, Destination: module.Root})
		if err != nil {
			t.Fatalf("generate report %s: %v", component.name, err)
		}
		declared := strings.TrimPrefix(strings.TrimSpace(compiled.Component.TypeContext.PackagePath), "/")
		want := module.ModulePath + "/" + declared
		if generated.Package.PkgPath != want || generated.Package.Name != filepath.Base(declared) {
			t.Fatalf("report %s generated package = %s (%s), want %s (%s)", component.name, generated.Package.PkgPath, generated.Package.Name, want, filepath.Base(declared))
		}
		if _, err = os.Stat(filepath.Join(module.Root, filepath.FromSlash(declared), "input.go")); err != nil {
			t.Fatalf("report %s input shape: %v", component.name, err)
		}
	}
	module.Test(t)
}

func compileReport(t *testing.T, ctx context.Context, component string, refiner *column.Refiner) *transcribe.Result {
	t.Helper()
	dir := component
	path := filepath.Join(dir, "report.dql")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	resources, err := resource.New().WithDefault(os.DirFS(dir))
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := transcribe.NewCompiler().Compile(ctx, &transcribe.Source{
		Scope: "github.com/viant/datly-studio/dql/studio/reports/" + component, Name: "report", Path: path,
		Text: string(payload), Connector: "studio", Resources: resources, ColumnRefiner: refiner, Types: datatest.StudioAuthorizationTypes(t),
	})
	if err != nil {
		t.Fatalf("compile report %s: %v", component, err)
	}
	return compiled
}
