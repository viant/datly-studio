package store_write_test

//go:generate env DATLY_GENERATE_ACL_STORE_WRITE=1 go test -run TestGenerateACLStoreWrite -count=1 .

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
	_ "modernc.org/sqlite"
)

func TestGenerateACLStoreWrite(t *testing.T) {
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
	directory := filepath.Join(root, "dql/studio/report_acl/store_write")
	payload, err := os.ReadFile(filepath.Join(directory, "acl.dql"))
	if err != nil {
		t.Fatal(err)
	}
	resources, err := resource.New().WithDefault(os.DirFS(directory))
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := transcribe.NewCompiler().Compile(ctx, &transcribe.Source{
		Scope: "github.com/viant/datly-studio/dql/studio/report_acl/store_write",
		Name:  "acl", Path: "acl.dql", Text: string(payload), Connector: "studio",
		Resources: resources, ColumnRefiner: column.New(column.Connections{"studio": db}),
	})
	if err != nil {
		t.Fatal(err)
	}
	destination := t.TempDir()
	if os.Getenv("DATLY_GENERATE_ACL_STORE_WRITE") == "1" {
		destination = root
	} else if err := os.WriteFile(filepath.Join(destination, "go.mod"), []byte("module github.com/viant/datly-studio\n\ngo 1.25.8\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (transcribe.Generator{Operation: "patch", EphemeralOwnership: true}).Generate(ctx,
		transcribe.GenerationRequest{Compiled: compiled, Destination: destination}); err != nil {
		t.Fatal(err)
	}
	packageDir := filepath.Join(destination, "studio/report_acl/store_write")
	views, err := os.ReadFile(filepath.Join(packageDir, "views.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"ReportId", "SubjectType", "SubjectId"} {
		if !regexp.MustCompile(`(?m)\b` + field + `\s+\*string\b`).Match(views) {
			t.Fatalf("generated ACL %s lost its pointer-valued key type", field)
		}
	}
	router, err := os.ReadFile(filepath.Join(packageDir, "router.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(router), "internal=true") {
		t.Fatal("generated ACL writer route is not internal")
	}
}
