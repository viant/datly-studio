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

// The import store writer inserts a version and its resource files as one
// aggregate: composite keys, the file relation and the ImportRules hook are
// the whole contract, and the route is in-process dispatch metadata only.
func TestVersionImportStoreWriterContract(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "import_store", "studio")
	compiled := compileVersionStore(t, ctx, "store_import", "version", column.New(column.Connections{"studio": db}))
	if err := column.ApplyWriterMetadata(compiled.Component); err != nil {
		t.Fatal(err)
	}
	if err := (bootstraproutes.Compiler{Component: compiled.Component}).Compile(); err != nil {
		t.Fatal(err)
	}
	routes := map[string]bool{}
	for _, route := range compiled.Component.Routes {
		routes[route.Method+" "+route.Path] = true
		if len(route.MCP) != 0 {
			t.Fatalf("server-only import writer must not be an MCP exposure: %+v", route.MCP)
		}
	}
	if len(routes) != 1 || !routes["POST /_studio/report-version-store/import"] {
		t.Fatalf("import writer routes = %#v", routes)
	}
	view := compiled.Component.RootView
	if view.EntityHooks != "ImportRules" {
		t.Fatalf("lifecycle hooks=%q", view.EntityHooks)
	}
	keys := map[string]bool{}
	for _, field := range view.Columns {
		if field != nil && field.PrimaryKey {
			keys[strings.ToLower(field.Name)] = true
		}
		if field != nil && field.ConcurrencyToken {
			t.Fatalf("a freshly inserted version has no concurrency token: %s", field.Name)
		}
	}
	if len(keys) != 2 || !keys["report_id"] || !keys["version_no"] {
		t.Fatalf("version keys=%v", keys)
	}
	if len(view.Relations) != 1 || view.Relations[0] == nil || view.Relations[0].View == nil || !strings.EqualFold(view.Relations[0].View.Name, "file") {
		t.Fatalf("file relation=%+v", view.Relations)
	}
	fileKeys := map[string]bool{}
	for _, field := range view.Relations[0].View.Columns {
		if field != nil && field.PrimaryKey {
			fileKeys[strings.ToLower(field.Name)] = true
		}
	}
	if len(fileKeys) != 3 || !fileKeys["report_id"] || !fileKeys["version_no"] || !fileKeys["resource_id"] {
		t.Fatalf("file keys=%v", fileKeys)
	}
	if source := view.Source; source == nil || len(source.Embeds) != 1 || source.Embeds[0].Path != "sql/read.sql" {
		t.Fatalf("version embedded SQL = %#v", source)
	}
	if source := view.Relations[0].View.Source; source == nil || len(source.Embeds) != 1 || source.Embeds[0].Path != "sql/files.sql" {
		t.Fatalf("file embedded SQL = %#v", source)
	}
}

// The head reader is the version allocator's only read: one required report
// scope, one aggregate row, no exposure.
func TestVersionHeadStoreReaderContract(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "head_store", "studio")
	compiled := compileVersionStore(t, ctx, "store_head", "head", column.New(column.Connections{"studio": db}))
	if err := (bootstraproutes.Compiler{Component: compiled.Component}).Compile(); err != nil {
		t.Fatal(err)
	}
	routes := map[string]bool{}
	for _, route := range compiled.Component.Routes {
		routes[route.Method+" "+route.Path] = true
		if len(route.MCP) != 0 {
			t.Fatalf("server-only head reader must not be an MCP exposure: %+v", route.MCP)
		}
	}
	if len(routes) != 1 || !routes["GET /_studio/report-version-store/head"] {
		t.Fatalf("head reader routes = %#v", routes)
	}
	required := 0
	for _, parameter := range compiled.Component.Parameters {
		if parameter == nil || parameter.Source.Kind != "query" {
			continue
		}
		if parameter.Name != "ReportId" || parameter.Required == nil || !*parameter.Required {
			t.Fatalf("head reader must require exactly the report scope: %+v", parameter)
		}
		required++
	}
	if required != 1 {
		t.Fatalf("head reader query parameters = %d", required)
	}
	view := compiled.Component.RootView
	if view.Source == nil || view.Source.Controls == nil || view.Source.Controls.Limit == nil || *view.Source.Controls.Limit != 1 {
		t.Fatalf("head reader must return one aggregate row: %+v", view.Source)
	}
	if len(view.Columns) != 1 || !strings.EqualFold(view.Columns[0].Name, "max_version_no") {
		t.Fatalf("head reader columns = %+v", view.Columns)
	}
}

func TestVersionStoreComponentsTranscribeDeclaredPackages(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "store_compile", "studio")
	module := datatest.NewGeneratedModule(t)
	for _, component := range []struct{ directory, name, operation string }{{"store_head", "head", "get"}, {"store_import", "version", "post"}} {
		compiled := compileVersionStore(t, ctx, component.directory, component.name, column.New(column.Connections{"studio": db}))
		generated, err := (transcribe.Generator{Operation: component.operation, EphemeralOwnership: true}).Generate(ctx, transcribe.GenerationRequest{Compiled: compiled, Destination: module.Root})
		if err != nil {
			t.Fatalf("generate %s: %v", component.directory, err)
		}
		declared := strings.TrimPrefix(strings.TrimSpace(compiled.Component.TypeContext.PackagePath), "/")
		want := module.ModulePath + "/" + declared
		if generated.Package.PkgPath != want || generated.Package.Name != filepath.Base(declared) {
			t.Fatalf("%s generated package = %s (%s), want %s", component.directory, generated.Package.PkgPath, generated.Package.Name, want)
		}
		for _, file := range []string{"input.go", "output.go", "views.go"} {
			if _, err = os.Stat(filepath.Join(module.Root, filepath.FromSlash(declared), file)); err != nil {
				t.Fatalf("%s %s: %v", component.directory, file, err)
			}
		}
	}
	module.Test(t)
}

func compileVersionStore(t *testing.T, ctx context.Context, directory, name string, refiner *column.Refiner) *transcribe.Result {
	t.Helper()
	path := filepath.Join(directory, name+".dql")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	resources, err := resource.New().WithDefault(os.DirFS(directory))
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := transcribe.NewCompiler().Compile(ctx, &transcribe.Source{
		Scope: "github.com/viant/datly-studio/dql/studio/report_versions/" + directory, Name: name, Path: path,
		Text: string(payload), Connector: "studio", Resources: resources, ColumnRefiner: refiner,
	})
	if err != nil {
		t.Fatalf("compile %s: %v", directory, err)
	}
	return compiled
}
