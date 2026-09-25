package store_write_test

import (
	"context"
	"os"
	"testing"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
)

func TestServerOwnedBFFSessionWriterCompiles(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "bff_store_writer", "studio")
	payload, err := os.ReadFile("session.dql")
	if err != nil {
		t.Fatal(err)
	}
	resources, err := resource.New().WithDefault(os.DirFS("."))
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := transcribe.NewCompiler().Compile(ctx, &transcribe.Source{
		Scope: "github.com/viant/datly-studio/dql/studio/bff_sessions/store_write", Name: "session", Path: "session.dql",
		Text: string(payload), Connector: "studio", Resources: resources,
		ColumnRefiner: column.New(column.Connections{"studio": db}),
	})
	if err != nil {
		t.Fatal(err)
	}
	module := datatest.NewGeneratedModule(t)
	if _, err := (transcribe.Generator{Operation: "patch", EphemeralOwnership: true}).Generate(ctx, transcribe.GenerationRequest{Compiled: compiled, Destination: module.Root}); err != nil {
		t.Fatal(err)
	}
	module.Test(t)
}
