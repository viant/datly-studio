package studio_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/viant/bindly/resource"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly/transcribe"
	"github.com/viant/datly/transcribe/column"
)

type staticComponent struct {
	path      string
	operation string
}

func TestGeneratedPackagesMatchCanonicalInventory(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	canonical := map[string]bool{
		"auth":       true,
		"connectors": true, "namespaces": true, "reports": true, "report_versions": true, "report_views": true,
		"report_parameters": true, "report_cube_configs": true, "report_mcp_exposures": true,
		"report_resource_files": true, "report_resource_folders": true, "report_skill_roots": true,
		"runtime_generations": true, "report_warmup_runs": true, "report_publications": true, "report_acl": true,
		"report_publication_events": true,
		"bff_sessions": true,
		"authorization_predicates": true,
	}
	legacy := map[string]bool{"report_filters": true, "components": true, "component_versions": true}
	support := map[string]bool{"authorization": true, "host": true, "predicatecatalog": true}
	for root, label := range map[string]string{filepath.Join(repoRoot, "studio"): "generated Go", filepath.Join(repoRoot, "dql", "studio"): "DQL"} {
		entries, err := os.ReadDir(root)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			if legacy[entry.Name()] {
				t.Fatalf("%s contains removed component directory %q", label, entry.Name())
			}
			if label == "generated Go" && support[entry.Name()] {
				continue
			}
			if !canonical[entry.Name()] {
				t.Errorf("%s contains unregistered component directory %q", label, entry.Name())
			}
		}
	}

	payload, err := os.ReadFile(filepath.Join(repoRoot, "schema", "schema.ddl"))
	if err != nil {
		t.Fatal(err)
	}
	tablePattern := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+([A-Za-z_][A-Za-z0-9_]*)`)
	for _, match := range tablePattern.FindAllStringSubmatch(string(payload), -1) {
		table := strings.ToLower(match[1])
		if table == "schema_version" {
			continue
		}
		if table == "report_fields" || table == "report_predicates" {
			continue
		}
		if !canonical[table] {
			t.Errorf("canonical table %q has no generated component package", table)
		}
	}
}

func TestGeneratedJSONColumnsRetainJSONEncoding(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	want := map[string][]string{
		"studio/connectors/reader/views.go":          {"OptionsJson"},
		"studio/connectors/writer/views.go":          {"OptionsJson"},
		"studio/report_versions/reader/views.go":     {"ComponentSpecJson", "DqlExportLimitsJson", "TypeManifestJson", "ResourceManifestJson", "ComponentDescriptorJson", "CompileDiagnosticsJson"},
		"studio/report_versions/writer/views.go":     {"ComponentSpecJson", "DqlExportLimitsJson", "TypeManifestJson", "ResourceManifestJson", "ComponentDescriptorJson", "CompileDiagnosticsJson"},
		"studio/report_views/reader/views.go":        {"MetadataJson"},
		"studio/report_parameters/reader/views.go":   {"QuerySelectorJson", "CodecJson", "ActivationJson", "MetadataJson", "ArgsJson"},
		"studio/report_parameters/writer/views.go":   {"QuerySelectorJson", "CodecJson", "ActivationJson", "MetadataJson", "ArgsJson"},
		"studio/report_cube_configs/reader/views.go": {"DimensionsJson", "MeasuresJson", "FiltersJson", "OrderByJson", "InputLayoutJson"},
		"studio/report_cube_configs/writer/views.go": {"DimensionsJson", "MeasuresJson", "FiltersJson", "OrderByJson", "InputLayoutJson"},
		"studio/runtime_generations/reader/views.go": {"BuildManifestJson", "DiagnosticsJson"},
		"studio/runtime_generations/writer/views.go": {"BuildManifestJson", "DiagnosticsJson"},
		"studio/report_warmup_runs/reader/views.go":  {"TargetJson", "DiagnosticsJson"},
		"studio/report_publications/reader/views.go": {"FailureJson"},
		"studio/report_publications/writer/views.go": {"FailureJson"},
	}
	for relative, fields := range want {
		payload, err := os.ReadFile(filepath.Join(repoRoot, relative))
		if err != nil {
			t.Fatal(err)
		}
		text := string(payload)
		for _, field := range fields {
			needle := field + ` json.RawMessage ` + "`sqlx:\""
			if !strings.Contains(text, needle) {
				t.Fatalf("%s is missing JSON raw field %s", relative, field)
			}
			position := strings.Index(text, needle)
			lineEnd := strings.Index(text[position:], "\n")
			line := text[position:]
			if lineEnd >= 0 {
				line = line[:lineEnd]
			}
			if !strings.Contains(line, "enc=JSON") {
				t.Fatalf("%s field %s is missing enc=JSON: %s", relative, field, line)
			}
		}
	}
}

// TestEveryStaticDatlyComponentContract keeps the canonical schema-backed
// component inventory honest. It compiles and generates every reader/writer
// package into one isolated Go module; runtime behavior is covered by the
// focused aggregate tests where the component has meaningful mutation policy.
func TestEveryStaticDatlyComponentContract(t *testing.T) {
	components := []staticComponent{
		{"auth/reader/context.dql", "get"},
		{"connectors/reader/connector.dql", "get"}, {"connectors/writer/connector.dql", "patch"},
		{"namespaces/reader/namespace.dql", "get"}, {"namespaces/writer/namespace.dql", "patch"},
		{"reports/reader/report.dql", "get"}, {"reports/writer/report.dql", "patch"},
		{"report_versions/reader/version.dql", "get"}, {"report_versions/writer/version.dql", "patch"},
		{"report_views/reader/view.dql", "get"},
		{"report_parameters/reader/parameter.dql", "get"}, {"report_parameters/writer/parameter.dql", "patch"},
		{"report_cube_configs/reader/config.dql", "get"}, {"report_cube_configs/writer/config.dql", "patch"},
		{"report_mcp_exposures/reader/exposure.dql", "get"}, {"report_mcp_exposures/writer/exposure.dql", "patch"},
		{"report_resource_files/reader/file.dql", "get"}, {"report_resource_files/writer/file.dql", "patch"},
		{"report_resource_folders/reader/folder.dql", "get"}, {"report_resource_folders/writer/folder.dql", "patch"},
		{"report_skill_roots/reader/skill.dql", "get"}, {"report_skill_roots/writer/skill.dql", "patch"},
		{"runtime_generations/reader/generation.dql", "get"}, {"runtime_generations/writer/generation.dql", "patch"},
		{"report_warmup_runs/reader/warmup_run.dql", "get"},
		{"report_publication_events/reader/event.dql", "get"},
		{"bff_sessions/reader/session.dql", "get"}, {"bff_sessions/writer/session.dql", "patch"},
		{"report_publications/reader/publication.dql", "get"}, {"report_publications/writer/publication.dql", "patch"},
		{"report_acl/reader/acl.dql", "get"}, {"report_acl/writer/acl.dql", "patch"},
	}
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "component_contract", "studio")
	module := datatest.NewGeneratedModule(t)
	types := datatest.StudioAuthorizationTypes(t)
	for _, item := range components {
		item := item
		t.Run(strings.TrimSuffix(item.path, ".dql"), func(t *testing.T) {
			directory := filepath.Dir(item.path)
			payload, err := os.ReadFile(item.path)
			if err != nil {
				t.Fatal(err)
			}
			for _, required := range []string{
				"#import('github.com/viant/scy/auth/jwt')",
				"$Jwt<string,*jwt.Claims>(header/Authorization)",
				"WithCodec(JwtClaim)",
				"WithStatusCode(401)",
			} {
				if !strings.Contains(string(payload), required) {
					t.Fatalf("component %s is missing authorization contract %q", item.path, required)
				}
			}
			if strings.Contains(string(payload), "WithPredicate(99, 'handler', 'authorization.") || strings.Contains(string(payload), "$predicate.FilterGroup(99") {
				t.Fatalf("component %s incorrectly couples JWT authentication to a row predicate", item.path)
			}
			if item.operation == "patch" && item.path != "connectors/writer/connector.dql" && item.path != "namespaces/writer/namespace.dql" && item.path != "reports/writer/report.dql" {
				if !strings.Contains(string(payload), "lifecycle_type(") {
					t.Fatalf("writer %s is missing its parent authorization lifecycle", item.path)
				}
			}
			generatedInput, err := os.ReadFile(filepath.Join("..", "..", "studio", directory, "input.go"))
			if err != nil {
				t.Fatalf("read generated input for %s: %v", item.path, err)
			}
			for _, required := range []string{
				"*jwt.Claims",
				"in=Authorization,dataType=string,errorCode=401,required=true",
				`codec:"JwtClaim"`,
			} {
				if !strings.Contains(string(generatedInput), required) {
					t.Fatalf("generated input for %s is missing authorization contract %q", item.path, required)
				}
			}
			if strings.Contains(string(generatedInput), `predicate:"handler,group=99,github.com/viant/datly-studio/studio/authorization.`) {
				t.Fatalf("generated input for %s incorrectly couples JWT authentication to a row predicate", item.path)
			}
			if item.operation == "patch" && item.path != "connectors/writer/connector.dql" && item.path != "namespaces/writer/namespace.dql" && item.path != "reports/writer/report.dql" {
				if !strings.Contains(string(generatedInput), "entityHooks=") {
					t.Fatalf("generated writer input for %s is missing lifecycle hook metadata", item.path)
				}
			}
			resources, err := resource.New().WithDefault(os.DirFS(directory))
			if err != nil {
				t.Fatal(err)
			}
			packagePath := "github.com/viant/datly-studio/dql/studio/" + directory
			compiled, err := transcribe.NewCompiler().Compile(ctx, &transcribe.Source{
				Scope: packagePath, Name: strings.TrimSuffix(filepath.Base(item.path), ".dql"), Path: item.path,
				Text: string(payload), Connector: "studio", Resources: resources,
				ColumnRefiner: column.New(column.Connections{"studio": db}), Types: types,
			})
			if err != nil {
				t.Fatal(err)
			}
			generated, err := (transcribe.Generator{Operation: item.operation, EphemeralOwnership: true}).Generate(ctx, transcribe.GenerationRequest{Compiled: compiled, Destination: module.Root})
			if err != nil {
				t.Fatal(err)
			}
			declared := strings.TrimPrefix(strings.TrimSpace(compiled.Component.TypeContext.PackagePath), "/")
			if generated.Package.PkgPath != module.ModulePath+"/"+declared {
				t.Fatalf("generated package=%s want=%s", generated.Package.PkgPath, module.ModulePath+"/"+declared)
			}
		})
	}
	module.Test(t)
}
