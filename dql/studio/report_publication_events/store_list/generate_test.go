package store_list_test

//go:generate env DATLY_GENERATE_PUBLICATION_EVENTS=1 go test -run TestGeneratePublicationEventReaders -count=1 .

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

func TestGeneratePublicationEventReaders(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = schema.ApplySQLite(ctx, db, "studio"); err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("../../../../")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct{ directory, name string }{
		{"store_list", "event"}, {"store_owner", "report"},
	} {
		directory := filepath.Join(root, "dql/studio/report_publication_events", item.directory)
		payload, readErr := os.ReadFile(filepath.Join(directory, item.name+".dql"))
		if readErr != nil {
			t.Fatal(readErr)
		}
		resources, resourceErr := resource.New().WithDefault(os.DirFS(directory))
		if resourceErr != nil {
			t.Fatal(resourceErr)
		}
		compiled, compileErr := transcribe.NewCompiler().Compile(ctx, &transcribe.Source{
			Scope: "github.com/viant/datly-studio/dql/studio/report_publication_events/" + item.directory,
			Name:  item.name, Path: item.name + ".dql", Text: string(payload), Connector: "studio",
			Resources: resources, ColumnRefiner: column.New(column.Connections{"studio": db}),
		})
		if compileErr != nil {
			t.Fatal(compileErr)
		}
		destination := t.TempDir()
		if os.Getenv("DATLY_GENERATE_PUBLICATION_EVENTS") == "1" {
			destination = root
		} else if writeErr := os.WriteFile(filepath.Join(destination, "go.mod"), []byte("module github.com/viant/datly-studio\n\ngo 1.25.8\n"), 0o600); writeErr != nil {
			t.Fatal(writeErr)
		}
		if _, generateErr := (transcribe.Generator{Operation: "get", EphemeralOwnership: true}).Generate(ctx,
			transcribe.GenerationRequest{Compiled: compiled, Destination: destination}); generateErr != nil {
			t.Fatal(generateErr)
		}
	}
}
