package store_catalog_test

//go:generate env DATLY_GENERATE_REPORT_CATALOG=1 go test -run TestGenerateReportCatalog -count=1 .

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
	_ "modernc.org/sqlite"
)

func TestGenerateReportCatalog(t *testing.T) {
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
	for _, name := range []string{"store_catalog", "get", "reader"} {
		directory := filepath.Join(root, "dql/studio/reports", name)
		payload, err := os.ReadFile(filepath.Join(directory, "report.dql"))
		if err != nil {
			t.Fatal(err)
		}
		resources, err := resource.New().WithDefault(os.DirFS(directory))
		if err != nil {
			t.Fatal(err)
		}
		compiled, err := transcribe.NewCompiler().Compile(ctx, &transcribe.Source{
			Scope: "github.com/viant/datly-studio/dql/studio/reports/" + name,
			Name:  "report", Path: "report.dql", Text: string(payload), Connector: "studio",
			Resources: resources, ColumnRefiner: column.New(column.Connections{"studio": db}), Types: datatest.StudioAuthorizationTypes(t),
		})
		if err != nil {
			t.Fatal(err)
		}
		destination := t.TempDir()
		if err := os.WriteFile(filepath.Join(destination, "go.mod"), []byte("module github.com/viant/datly-studio\n\ngo 1.25.8\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := (transcribe.Generator{Operation: "get"}).Generate(ctx,
			transcribe.GenerationRequest{Compiled: compiled, Destination: destination}); err != nil {
			t.Fatal(err)
		}
		if os.Getenv("DATLY_GENERATE_REPORT_CATALOG") == "1" {
			// These older public packages retain hand-authored lifecycle and binding files.
			// Refresh only generated projections and SQL from a clean generation tree.
			base := filepath.Join(destination, "studio/reports", name)
			for _, file := range []string{"views.go", "output.go", "sql/read.sql", "sql/report.sql"} {
				payload, err := os.ReadFile(filepath.Join(base, file))
				if os.IsNotExist(err) {
					continue
				}
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(filepath.Join(root, "studio/reports", name, file), payload, 0600); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
