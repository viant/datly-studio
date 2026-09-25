package store_read_test

import (
	"context"
	"os"
	"testing"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
)

func TestServerOwnedBFFSessionReaderCompiles(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "bff_store_reader", "studio")
	payload, err := os.ReadFile("session.dql")
	if err != nil {
		t.Fatal(err)
	}
	resources, err := resource.New().WithDefault(os.DirFS("."))
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := transcribe.NewCompiler().Compile(ctx, &transcribe.Source{
		Scope: "github.com/viant/datly-studio/dql/studio/bff_sessions/store_read", Name: "session", Path: "session.dql",
		Text: string(payload), Connector: "studio", Resources: resources,
		ColumnRefiner: column.New(column.Connections{"studio": db}),
	})
	if err != nil {
		t.Fatal(err)
	}
	identities := 0
	for _, parameter := range compiled.Component.Parameters {
		if parameter == nil || parameter.Source.Kind != "query" {
			continue
		}
		if parameter.Required == nil || !*parameter.Required || len(parameter.Predicates) != 1 {
			t.Fatalf("session identity is not a required predicate: %+v", parameter)
		}
		identities++
	}
	if identities != 1 {
		t.Fatalf("session identity predicate count=%d", identities)
	}
	module := datatest.NewGeneratedModule(t)
	if _, err := (transcribe.Generator{Operation: "get", EphemeralOwnership: true}).Generate(ctx, transcribe.GenerationRequest{Compiled: compiled, Destination: module.Root}); err != nil {
		t.Fatal(err)
	}
	module.Test(t)
}
