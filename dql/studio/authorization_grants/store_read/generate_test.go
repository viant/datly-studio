package store_read_test

//go:generate env DATLY_GENERATE_AUTHORIZATION_GRANTS=1 go test -run TestGenerateAuthorizationGrants -count=1 .

import (
	"bytes"
	"context"
	"database/sql"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/schema"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
	"github.com/viant/sqlparser"
	"github.com/viant/sqlparser/expr"
	_ "modernc.org/sqlite"
)

func TestGenerateAuthorizationGrants(t *testing.T) {
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
	directory := filepath.Join(root, "dql/studio/authorization_grants/store_read")
	payload, err := os.ReadFile(filepath.Join(directory, "grant.dql"))
	if err != nil {
		t.Fatal(err)
	}
	resources, err := resource.New().WithDefault(os.DirFS(directory))
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := transcribe.NewCompiler().Compile(ctx, &transcribe.Source{
		Scope: "github.com/viant/datly-studio/dql/studio/authorization_grants/store_read",
		Name:  "grant", Path: "grant.dql", Text: string(payload), Connector: "studio",
		Resources: resources, ColumnRefiner: column.New(column.Connections{"studio": db}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if compiled.Component.Name != "grant" || compiled.Component.RootView.Name != "grant" || compiled.Component.RootView.Namespace != "grant_rows" || len(compiled.Component.Routes) != 1 || compiled.Component.Routes[0].Path != "/_studio/authorization-grant-store/read" {
		t.Fatalf("logical component=%q root=%q SQL namespace=%q route count=%d; expected grant/grant/grant_rows and unchanged route", compiled.Component.Name, compiled.Component.RootView.Name, compiled.Component.RootView.Namespace, len(compiled.Component.Routes))
	}
	wantFields := []string{"can_edit", "can_publish", "can_view", "grant_source", "is_live", "report_id", "subject_id", "subject_type"}
	gotFields := make([]string, 0, len(compiled.Component.RootView.Columns))
	for _, field := range compiled.Component.RootView.Columns {
		name := field.Source
		if name == "" {
			name = field.Name
		}
		name = strings.Trim(name, "\"`")
		if i := strings.LastIndex(name, "."); i >= 0 {
			name = name[i+1:]
		}
		gotFields = append(gotFields, name)
	}
	sort.Strings(gotFields)
	if strings.Join(gotFields, ",") != strings.Join(wantFields, ",") {
		t.Fatalf("public eight-field grant projection changed: %v", gotFields)
	}
	innerSQL, err := os.ReadFile(filepath.Join(directory, "sql/read.sql"))
	if err != nil {
		t.Fatal(err)
	}
	// The predicate suffix has no projection authority; parse the complete
	// fixed inner SELECT before that known runtime predicate insertion point.
	inner, err := sqlparser.ParseQuery(strings.Split(string(innerSQL), "${predicate.Builder")[0])
	if err != nil {
		t.Fatal(err)
	}
	innerFields := []string{}
	for _, field := range inner.List {
		selector, ok := field.Expr.(*expr.Selector)
		if !ok {
			t.Fatalf("unexpected inner projection %T", field.Expr)
		}
		leaf, ok := selector.X.(*expr.Ident)
		if !ok || selector.Name != "g" || selector.Expression != "" {
			t.Fatalf("unexpected qualified inner field %#v", selector)
		}
		innerFields = append(innerFields, leaf.Name)
	}
	sort.Strings(innerFields)
	if strings.Join(innerFields, ",") != strings.Join(wantFields, ",") {
		t.Fatalf("derived wildcard would expose changed inner fields: %v", innerFields)
	}
	destination := t.TempDir()
	if os.Getenv("DATLY_GENERATE_AUTHORIZATION_GRANTS") == "1" {
		destination = root
	} else if err := os.WriteFile(filepath.Join(destination, "go.mod"), []byte("module github.com/viant/datly-studio\n\ngo 1.25.8\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (transcribe.Generator{Operation: "get"}).Generate(ctx,
		transcribe.GenerationRequest{Compiled: compiled, Destination: destination}); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("DATLY_GENERATE_AUTHORIZATION_GRANTS") != "1" {
		generatedSQL, err := os.ReadFile(filepath.Join(destination, "studio/authorization_grants/store_read/sql/read.sql"))
		if err != nil {
			t.Fatal(err)
		}
		runtimeSQL, err := os.ReadFile(filepath.Join(root, "studio/authorization_grants/store_read/sql/read.sql"))
		if err != nil {
			t.Fatal(err)
		}
		if evidence := os.Getenv("DATLY_GRANT_GENERATION_EVIDENCE"); evidence != "" {
			if err := os.MkdirAll(evidence, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(evidence, "generated.sql"), generatedSQL, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(evidence, "runtime.sql"), runtimeSQL, 0600); err != nil {
				t.Fatal(err)
			}
		}
		if !bytes.Equal(bytes.TrimSpace(generatedSQL), bytes.TrimSpace(runtimeSQL)) {
			t.Fatal("canonical regeneration does not match the runtime SQL resource")
		}
		baseline := publicGrantAPI(t, filepath.Join(root, "studio/authorization_grants/store_read"))
		generated := publicGrantAPI(t, filepath.Join(destination, "studio/authorization_grants/store_read"))
		if baseline != generated {
			t.Fatalf("SQL alias correction changed public Go shapes/signatures\nbaseline:%s\ngenerated:%s", baseline, generated)
		}
	}
}

// Tags carry private SQL graph names; compare public declarations separately.
func publicGrantAPI(t *testing.T, directory string) string {
	t.Helper()
	fset := token.NewFileSet()
	packages, err := parser.ParseDir(fset, directory, func(info os.FileInfo) bool { return !strings.HasSuffix(info.Name(), "_test.go") }, 0)
	if err != nil {
		t.Fatal(err)
	}
	var declarations []string
	for _, pkg := range packages {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				switch n := decl.(type) {
				case *ast.GenDecl:
					for _, spec := range n.Specs {
						if typ, ok := spec.(*ast.TypeSpec); ok && typ.Name.IsExported() {
							if fields, ok := typ.Type.(*ast.StructType); ok {
								for _, field := range fields.Fields.List {
									field.Tag = nil
								}
							}
							var b bytes.Buffer
							printer.Fprint(&b, token.NewFileSet(), typ)
							declarations = append(declarations, "type "+b.String())
						}
					}
				case *ast.FuncDecl:
					if n.Name.IsExported() {
						n.Body = nil
						var b bytes.Buffer
						printer.Fprint(&b, token.NewFileSet(), n)
						declarations = append(declarations, b.String())
					}
				}
			}
		}
	}
	sort.Strings(declarations)
	return strings.Join(declarations, "\n")
}
