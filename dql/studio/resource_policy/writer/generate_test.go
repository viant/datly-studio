package writer_test

//go:generate env DATLY_GENERATE_POLICY_AUDIT=1 go test -run TestGeneratePolicyAudit -count=1 .

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
	_ "modernc.org/sqlite"
)

func TestGeneratePolicyAudit(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("../../../../")
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, "dql/studio/resource_policy/writer")
	payload, err := os.ReadFile(filepath.Join(directory, "policy.dql"))
	if err != nil {
		t.Fatal(err)
	}
	resources, err := resource.New().WithDefault(os.DirFS(directory))
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := transcribe.NewCompiler().Compile(ctx, &transcribe.Source{
		Scope: "github.com/viant/datly-studio/dql/studio/resource_policy/writer",
		Name:  "policy", Path: "policy.dql", Text: string(payload), Connector: "studio",
		Resources: resources, ColumnRefiner: column.New(column.Connections{"studio": db}),
	})
	if err != nil {
		t.Fatal(err)
	}
	destination := t.TempDir()
	if os.Getenv("DATLY_GENERATE_POLICY_AUDIT") == "1" {
		destination = root
	} else if err := os.WriteFile(filepath.Join(destination, "go.mod"), []byte("module github.com/viant/datly-studio\n\ngo 1.25.8\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (transcribe.Generator{Operation: "patch", EphemeralOwnership: true}).Generate(ctx,
		transcribe.GenerationRequest{Compiled: compiled, Destination: destination}); err != nil {
		t.Fatal(err)
	}
	if destination == root {
		// This component runs through Datly's universal writer. Keep only
		// transcribed metadata; generated phase executors are intentionally unused.
		for _, name := range []string{"actions.go", "frames.go", "layout.go", "links.go", "mutation.go",
			"mutation_output.go", "previous.go", "validation.go", "lifecycle_dispatch.go"} {
			err := os.Remove(filepath.Join(root, "studio/resource_policy/writer", name))
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				t.Fatal(err)
			}
		}
	}
}
