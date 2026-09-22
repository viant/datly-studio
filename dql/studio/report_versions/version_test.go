package reportversions_test

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

func TestReportVersionReaderContract(t *testing.T) {
	compiled := compileVersion(t, context.Background(), "reader", nil)
	if err := (bootstraproutes.Compiler{Component: compiled.Component}).Compile(); err != nil {
		t.Fatal(err)
	}
	routes, tools := map[string]bool{}, map[string]bool{}
	for _, route := range compiled.Component.Routes {
		routes[route.Method+" "+route.Path] = true
		for _, exposure := range route.MCP {
			if exposure != nil {
				tools[exposure.Name] = true
			}
		}
	}
	if !routes["GET /v1/studio/reports/{reportId}/versions"] || !routes["GET /v1/studio/reports/{reportId}/versions/{versionNo}"] {
		t.Fatalf("report-version routes = %#v", routes)
	}
	if !tools["studio.report_versions.read"] || !tools["studio.report_versions.readByVersionNo"] {
		t.Fatalf("report-version tools = %#v", tools)
	}
	if source := compiled.Component.RootView.Source; source == nil || len(source.Embeds) != 1 || source.Embeds[0].Path != "sql/read.sql" {
		t.Fatalf("report-version embedded SQL = %#v", source)
	}
}

func TestReportVersionWriterCompositeIdentityAndConcurrency(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "writer_compile", "studio")
	compiled := compileVersion(t, ctx, "writer", column.New(column.Connections{"studio": db}))
	if err := column.ApplyWriterMetadata(compiled.Component); err != nil {
		t.Fatal(err)
	}
	view := compiled.Component.RootView
	keys, concurrency := map[string]bool{}, false
	for _, field := range view.Columns {
		if field == nil {
			continue
		}
		if field.PrimaryKey {
			keys[strings.ToLower(field.Name)] = true
		}
		concurrency = concurrency || strings.EqualFold(field.Name, "source_revision") && field.ConcurrencyToken
	}
	if !keys["report_id"] || !keys["version_no"] || len(keys) != 2 || !concurrency {
		t.Fatalf("report-version mutation metadata: keys=%v concurrency=%v", keys, concurrency)
	}
}

func TestReportVersionTranscribesDeclaredPackages(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "runtime_compile", "studio")
	module := datatest.NewGeneratedModule(t)
	for _, component := range []struct{ name, operation string }{{"reader", "get"}, {"writer", "patch"}} {
		compiled := compileVersion(t, ctx, component.name, column.New(column.Connections{"studio": db}))
		generated, err := (transcribe.Generator{Operation: component.operation, EphemeralOwnership: true}).Generate(ctx, transcribe.GenerationRequest{Compiled: compiled, Destination: module.Root})
		if err != nil {
			t.Fatalf("generate report-version %s: %v", component.name, err)
		}
		declared := strings.TrimPrefix(strings.TrimSpace(compiled.Component.TypeContext.PackagePath), "/")
		want := module.ModulePath + "/" + declared
		if generated.Package.PkgPath != want || generated.Package.Name != filepath.Base(declared) {
			t.Fatalf("report-version %s generated package = %s (%s), want %s", component.name, generated.Package.PkgPath, generated.Package.Name, want)
		}
		if _, err = os.Stat(filepath.Join(module.Root, filepath.FromSlash(declared), "input.go")); err != nil {
			t.Fatalf("report-version %s input: %v", component.name, err)
		}
	}
	module.Test(t)
}

func compileVersion(t *testing.T, ctx context.Context, component string, refiner *column.Refiner) *transcribe.Result {
	t.Helper()
	dir := component
	path := filepath.Join(dir, "version.dql")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	resources, err := resource.New().WithDefault(os.DirFS(dir))
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := transcribe.NewCompiler().Compile(ctx, &transcribe.Source{
		Scope: "github.com/viant/datly-studio/dql/studio/report_versions/" + component, Name: "report_version", Path: path,
		Text: string(payload), Connector: "studio", Resources: resources, ColumnRefiner: refiner, Types: datatest.StudioAuthorizationTypes(t),
	})
	if err != nil {
		t.Fatalf("compile report-version %s: %v", component, err)
	}
	return compiled
}
