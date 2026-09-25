package runtime_generations_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
)

func TestServerOwnedDefinitionReadersCompile(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "runtime_definition_readers", "studio")
	module := datatest.NewGeneratedModule(t)
	for _, name := range []string{"store_active", "store_candidate"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(name, "definition.dql")
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			resources, err := resource.New().WithDefault(os.DirFS(name))
			if err != nil {
				t.Fatal(err)
			}
			compiled, err := transcribe.NewCompiler().Compile(ctx, &transcribe.Source{
				Scope: "github.com/viant/datly-studio/dql/studio/runtime_generations/" + name,
				Name:  "definition", Path: path, Text: string(body), Connector: "studio", Resources: resources,
				ColumnRefiner: column.New(column.Connections{"studio": db}),
			})
			if err != nil {
				t.Fatal(err)
			}
			if name == "store_candidate" && (!strings.Contains(string(body), ".Required()") || len(compiled.Component.Parameters) < 2) {
				t.Fatal("candidate generation must be a required component input")
			}
			if _, err := (transcribe.Generator{Operation: "get", EphemeralOwnership: true}).Generate(ctx,
				transcribe.GenerationRequest{Compiled: compiled, Destination: module.Root}); err != nil {
				t.Fatal(err)
			}
		})
	}
	module.Test(t)
}
