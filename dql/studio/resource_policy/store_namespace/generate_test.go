package store_namespace_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
	_ "modernc.org/sqlite"
)

func TestNamespacePolicyReaderGeneration(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"store_namespace"} {
		directory := filepath.Join(root, "dql/studio/resource_policy", kind)
		text, err := os.ReadFile(filepath.Join(directory, "namespace.dql"))
		if err != nil {
			t.Fatal(err)
		}
		resources, err := resource.New().WithDefault(os.DirFS(directory))
		if err != nil {
			t.Fatal(err)
		}
		compiled, err := transcribe.NewCompiler().Compile(ctx, &transcribe.Source{Scope: "github.com/viant/datly-studio/dql/studio/resource_policy/" + kind, Name: "namespace", Path: "namespace.dql", Text: string(text), Connector: "studio", Resources: resources, ColumnRefiner: column.New(column.Connections{"studio": db})})
		if err != nil {
			t.Fatal(err)
		}
		destination := t.TempDir()
		if err = os.WriteFile(filepath.Join(destination, "go.mod"), []byte("module github.com/viant/datly-studio\n\ngo 1.25.8\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if os.Getenv("DATLY_GENERATE_NAMESPACE_POLICY") == "1" {
			destination = root
		}
		operation := "get"
		if _, err = (transcribe.Generator{Operation: operation}).Generate(ctx, transcribe.GenerationRequest{Compiled: compiled, Destination: destination}); err != nil {
			t.Fatal(err)
		}
	}
}
