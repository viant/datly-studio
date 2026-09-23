package resourcepolicy_test

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

func TestResourcePolicyReaderContract(t *testing.T) {
	compiled := compilePolicy(t, context.Background(), "reader", nil)
	if err := (bootstraproutes.Compiler{Component: compiled.Component}).Compile(); err != nil {
		t.Fatal(err)
	}
	routes := map[string]bool{}
	for _, route := range compiled.Component.Routes {
		routes[route.Method+" "+route.Path] = true
		if len(route.MCP) != 0 {
			t.Fatalf("server-owned policy reader must not be an MCP exposure: %+v", route.MCP)
		}
	}
	if !routes["GET /v1/studio/resource-policies"] || len(routes) != 1 {
		t.Fatalf("resource policy routes = %#v", routes)
	}
	if source := compiled.Component.RootView.Source; source == nil || len(source.Embeds) != 1 || source.Embeds[0].Path != "sql/read.sql" {
		t.Fatalf("resource policy embedded SQL = %#v", source)
	}
	required := 0
	for _, parameter := range compiled.Component.Parameters {
		if parameter == nil || parameter.Source.Kind != "query" {
			continue
		}
		if parameter.Required == nil || !*parameter.Required || len(parameter.Predicates) != 1 {
			t.Fatalf("every identity dimension must be a required predicate: %+v", parameter)
		}
		required++
	}
	if required != 4 {
		t.Fatalf("identity dimensions = %d", required)
	}
}

func TestResourcePolicyWriterCompositeIdentityAndConcurrency(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "writer_compile", "studio")
	compiled := compilePolicy(t, ctx, "writer", column.New(column.Connections{"studio": db}))
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
		concurrency = concurrency || strings.EqualFold(field.Name, "revision") && field.ConcurrencyToken
	}
	if len(keys) != 4 || !keys["tenant_id"] || !keys["resource_kind"] || !keys["resource_id"] || !keys["resource_version"] || !concurrency {
		t.Fatalf("resource policy mutation metadata: keys=%v concurrency=%v", keys, concurrency)
	}
	if view.EntityHooks != "PolicyRules" {
		t.Fatalf("lifecycle hooks=%q", view.EntityHooks)
	}
	if len(view.Relations) != 1 || view.Relations[0] == nil || view.Relations[0].View == nil || !strings.EqualFold(view.Relations[0].View.Name, "history") {
		t.Fatalf("history relation=%+v", view.Relations)
	}
	historyKeys := map[string]bool{}
	for _, field := range view.Relations[0].View.Columns {
		if field != nil && field.PrimaryKey {
			historyKeys[strings.ToLower(field.Name)] = true
		}
	}
	if len(historyKeys) != 5 || !historyKeys["revision"] {
		t.Fatalf("history keys=%v", historyKeys)
	}
}

func TestResourcePolicyTranscribesDeclaredPackages(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "runtime_compile", "studio")
	module := datatest.NewGeneratedModule(t)
	for _, component := range []struct{ name, operation string }{{"reader", "get"}, {"writer", "patch"}} {
		compiled := compilePolicy(t, ctx, component.name, column.New(column.Connections{"studio": db}))
		generated, err := (transcribe.Generator{Operation: component.operation, EphemeralOwnership: true}).Generate(ctx, transcribe.GenerationRequest{Compiled: compiled, Destination: module.Root})
		if err != nil {
			t.Fatalf("generate resource policy %s: %v", component.name, err)
		}
		declared := strings.TrimPrefix(strings.TrimSpace(compiled.Component.TypeContext.PackagePath), "/")
		want := module.ModulePath + "/" + declared
		if generated.Package.PkgPath != want || generated.Package.Name != filepath.Base(declared) {
			t.Fatalf("resource policy %s generated package = %s (%s), want %s", component.name, generated.Package.PkgPath, generated.Package.Name, want)
		}
		for _, file := range []string{"input.go", "output.go", "views.go"} {
			if _, err = os.Stat(filepath.Join(module.Root, filepath.FromSlash(declared), file)); err != nil {
				t.Fatalf("resource policy %s %s: %v", component.name, file, err)
			}
		}
	}
	module.Test(t)
}

func compilePolicy(t *testing.T, ctx context.Context, component string, refiner *column.Refiner) *transcribe.Result {
	t.Helper()
	dir := component
	path := filepath.Join(dir, "policy.dql")
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	resources, err := resource.New().WithDefault(os.DirFS(dir))
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := transcribe.NewCompiler().Compile(ctx, &transcribe.Source{
		Scope: "github.com/viant/datly-studio/dql/studio/resource_policy/" + component, Name: "resource_policy", Path: path,
		Text: string(payload), Connector: "studio", Resources: resources, ColumnRefiner: refiner, Types: datatest.StudioAuthorizationTypes(t),
	})
	if err != nil {
		t.Fatalf("compile resource policy %s: %v", component, err)
	}
	return compiled
}
