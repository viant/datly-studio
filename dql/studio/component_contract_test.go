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
		"connectors": true, "namespaces": true, "components": true, "reports": true, "component_versions": true, "component_views": true,
		"component_parameters": true, "component_cube_configs": true, "component_mcp_exposures": true,
		"component_resource_files": true, "component_resource_folders": true, "component_skill_roots": true,
		"runtime_generations": true, "component_warmup_runs": true, "component_publications": true, "component_acl": true,
		"component_publication_events": true,
		"bff_sessions":                 true,
		"authorization_predicates":     true,
		"resource_policy":              true,
		"resource_namespaces":          true,
		"resource_namespace_claims":    true,
		"authorization_grants":         true,
	}
	legacy := map[string]bool{"report_filters": true}
	support := map[string]bool{"authorization": true, "host": true, "predicatecatalog": true, "resources": true, "preview": true, "runtime": true}
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
			// Package paths retain their public names; SQL tables use component names.
			directoryKey := strings.Replace(entry.Name(), "report_", "component_", 1)
			if !canonical[directoryKey] {
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
		if table == "component_fields" || table == "component_predicates" {
			continue
		}
		// This is a Studio migration sidecar that binds legacy shared policy
		// namespace metadata; it is not a runtime Datly component resource.
		if table == "resource_policy_namespace_bindings" {
			continue
		}
		if table == "resource_policies" || table == "resource_policy_revisions" {
			if !canonical["resource_policy"] {
				t.Errorf("canonical table %q has no resource_policy component package", table)
			}
			continue
		}
		if !canonical[table] {
			t.Errorf("canonical table %q has no generated component package", table)
		}
	}
}

func TestServerOnlyStoreComponentsRemainInProcessOnly(t *testing.T) {
	configuration, err := os.ReadFile(filepath.Join("..", "..", "datly.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"store_read", "store_write", "store_expired"} {
		path := "github.com/viant/datly-studio/studio/bff_sessions/" + name
		if strings.Contains(string(configuration), path) {
			t.Fatalf("server-only BFF component %s is exposed by the public Datly bootstrap", path)
		}
	}
	if path := "github.com/viant/datly-studio/studio/connectors/store_active"; strings.Contains(string(configuration), path) {
		t.Fatalf("server-only connector component %s is exposed by the public Datly bootstrap", path)
	}
	for _, path := range []string{
		"github.com/viant/datly-studio/studio/report_skill_roots/store_validation",
		"github.com/viant/datly-studio/studio/report_resource_files/store_skill_content",
		"github.com/viant/datly-studio/studio/runtime_generations/store_active",
		"github.com/viant/datly-studio/studio/runtime_generations/store_candidate",
		"github.com/viant/datly-studio/studio/runtime_generations/store_status",
		"github.com/viant/datly-studio/studio/runtime_generations/store_readers",
		"github.com/viant/datly-studio/studio/runtime_generations/store_head",
		"github.com/viant/datly-studio/studio/runtime_generations/store_insert",
		"github.com/viant/datly-studio/studio/runtime_generations/store_state",
		"github.com/viant/datly-studio/studio/runtime_generations/store_active_others",
		"github.com/viant/datly-studio/studio/report_versions/store_preview_definition",
		"github.com/viant/datly-studio/studio/connectors/store_preview_scoped",
		"github.com/viant/datly-studio/studio/connectors/store_insert",
		"github.com/viant/datly-studio/studio/connectors/store_runtime_catalog",
		"github.com/viant/datly-studio/studio/report_versions/store_state",
		"github.com/viant/datly-studio/studio/report_versions/store_mcp_names",
		"github.com/viant/datly-studio/studio/report_acl/store_list",
		"github.com/viant/datly-studio/studio/report_acl/store_one",
		"github.com/viant/datly-studio/studio/namespaces/store_insert",
		"github.com/viant/datly-studio/studio/namespaces/store_access",
		"github.com/viant/datly-studio/studio/authorization_grants/store_read",
		"github.com/viant/datly-studio/studio/report_publication_events/store_list",
		"github.com/viant/datly-studio/studio/report_publication_events/store_owner",
		"github.com/viant/datly-studio/studio/report_publication_events/store_insert",
		"github.com/viant/datly-studio/studio/report_publications/store_status",
		"github.com/viant/datly-studio/studio/report_publications/store_insert",
		"github.com/viant/datly-studio/studio/report_publications/store_stage",
		"github.com/viant/datly-studio/studio/report_publications/store_activate",
		"github.com/viant/datly-studio/studio/report_publications/store_active_others",
		"github.com/viant/datly-studio/studio/report_publications/store_repoint",
		"github.com/viant/datly-studio/studio/report_publications/store_runtime_catalog",
	} {
		if strings.Contains(string(configuration), path) {
			t.Fatalf("server-only component %s is exposed by the public Datly bootstrap", path)
		}
	}
}

func TestSelectedPrivateWorkflowChildrenStayInternalOnly(t *testing.T) {
	configuration, err := os.ReadFile(filepath.Join("..", "..", "datly.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct{ path, router string }{
		{"studio/report_versions/store_edit", "studio/report_versions/store_edit/router.go"},
		{"studio/report_versions/store_validation", "studio/report_versions/store_validation/router.go"},
		{"studio/report_versions/store_catalog", "studio/report_versions/store_catalog/router.go"},
		{"studio/report_versions/store_touch", "studio/report_versions/store_touch/router.go"},
		{"studio/report_versions/store_import", "studio/report_versions/store_import/router.go"},
		{"studio/reports/store_draft_pointer", "studio/reports/store_draft_pointer/router.go"},
		{"studio/report_acl/store_write", "studio/report_acl/store_write/router.go"},
		{"studio/reports/edit_guard", "studio/reports/edit_guard/router.go"},
		{"studio/reports/store_config", "studio/reports/store_config/router.go"},
		{"studio/report_versions/store_head", "studio/report_versions/store_head/router.go"},
		{"studio/report_versions/store_insert", "studio/report_versions/store_insert/router.go"},
		{"studio/reports/store_capabilities", "studio/reports/store_capabilities/router.go"},
		{"studio/reports/store_global_access", "studio/reports/store_global_access/router.go"},
		{"studio/reports/store_run_access", "studio/reports/store_run_access/router.go"},
		{"studio/authorization_predicates/store_read", "studio/authorization_predicates/store_read/router.go"},
		{"studio/authorization_predicates/store_insert", "studio/authorization_predicates/store_insert/router.go"},
		{"studio/authorization_predicates/store_write", "studio/authorization_predicates/store_write/router.go"},
		{"studio/connectors/store_access", "studio/connectors/store_access/router.go"},
		{"studio/connectors/store_catalog", "studio/connectors/store_catalog/router.go"},
		{"studio/connectors/store_config", "studio/connectors/store_config/router.go"},
		{"studio/connectors/store_status", "studio/connectors/store_status/router.go"},
		{"studio/connectors/store_usage", "studio/connectors/store_usage/router.go"},
		{"studio/namespaces/store_read", "studio/namespaces/store_read/router.go"},
		{"studio/namespaces/store_usage", "studio/namespaces/store_usage/router.go"},
		{"studio/namespaces/store_write", "studio/namespaces/store_write/router.go"},
		{"studio/reports/store_insert", "studio/reports/store_insert/router.go"},
		{"studio/reports/store_catalog", "studio/reports/store_catalog/router.go"},
		{"studio/report_resource_files/store_download", "studio/report_resource_files/store_download/router.go"},
		{"studio/report_resource_files/store_download_budget", "studio/report_resource_files/store_download_budget/router.go"},
		{"studio/report_resource_files/store_snapshot", "studio/report_resource_files/store_snapshot/router.go"},
		{"studio/report_resource_files/store_write", "studio/report_resource_files/store_write/router.go"},
		{"studio/report_resource_folders/store_snapshot", "studio/report_resource_folders/store_snapshot/router.go"},
		{"studio/report_resource_folders/store_write", "studio/report_resource_folders/store_write/router.go"},
		{"studio/report_skill_roots/store_snapshot", "studio/report_skill_roots/store_snapshot/router.go"},
		{"studio/report_skill_roots/store_write", "studio/report_skill_roots/store_write/router.go"},
		{"studio/resource_namespace_claims/store_write", "studio/resource_namespace_claims/store_write/router.go"},
		{"studio/resource_namespaces/store_presence", "studio/resource_namespaces/store_presence/router.go"},
		{"studio/resource_namespaces/store_usage", "studio/resource_namespaces/store_usage/router.go"},
		{"studio/reports/publish_guard", "studio/reports/publish_guard/router.go"},
		{"studio/report_warmup_runs/store_expired", "studio/report_warmup_runs/store_expired/router.go"},
		{"studio/report_warmup_runs/store_read", "studio/report_warmup_runs/store_read/router.go"},
		{"studio/report_warmup_runs/store_write", "studio/report_warmup_runs/store_write/router.go"},
	} {
		packagePath := "github.com/viant/datly-studio/" + item.path
		if !strings.Contains(string(configuration), packagePath) {
			t.Fatalf("native workflow child %s is not selected", packagePath)
		}
		metadata, readErr := os.ReadFile(filepath.Join("..", "..", item.router))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if !strings.Contains(string(metadata), "internal=true") || strings.Contains(string(metadata), ` mcp:"`) {
			t.Fatalf("selected child %s is not internal-only", packagePath)
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
			pattern := regexp.MustCompile(regexp.QuoteMeta(field) + `\s+json\.RawMessage\s+` + "`")
			location := pattern.FindStringIndex(text)
			if location == nil {
				t.Fatalf("%s is missing JSON raw field %s", relative, field)
			}
			position := location[0]
			lineEnd := strings.Index(text[position:], "\n")
			line := text[position:]
			if lineEnd >= 0 {
				line = line[:lineEnd]
			}
			if !strings.Contains(line, "sqlx:\"") || !strings.Contains(line, "enc=JSON") {
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
		{"connectors/get/connector.dql", "get"},
		{"connectors/store_active/connector.dql", "get"},
		{"connectors/store_preview_scoped/connector.dql", "get"},
		{"connectors/store_catalog/connector.dql", "get"},
		{"connectors/store_insert/connector.dql", "post"},
		{"connectors/store_usage/usage.dql", "get"},
		{"connectors/store_status/connector.dql", "patch"},
		{"connectors/store_config/connector.dql", "patch"},
		{"connectors/store_access/connector.dql", "get"},
		{"connectors/store_runtime_catalog/connector.dql", "get"},
		{"report_versions/store_head/head.dql", "get"},
		{"report_versions/store_catalog/version.dql", "get"},
		{"report_versions/store_insert/version.dql", "post"},
		{"report_versions/store_edit/version.dql", "patch"},
		{"report_versions/store_validation/version.dql", "patch"},
		{"report_versions/store_state/version.dql", "patch"},
		{"report_versions/store_mcp_names/version.dql", "get"},
		{"report_versions/store_import/version.dql", "post"},
		{"reports/store_draft_pointer/report.dql", "patch"},
		{"reports/store_insert/report.dql", "post"},
		{"reports/store_config/report.dql", "patch"},
		{"reports/store_global_access/report.dql", "get"},
		{"reports/store_catalog/report.dql", "get"},
		{"namespaces/reader/namespace.dql", "get"}, {"namespaces/writer/namespace.dql", "patch"},
		{"namespaces/get/namespace.dql", "get"},
		{"namespaces/store_read/namespace.dql", "get"}, {"namespaces/store_usage/usage.dql", "get"},
		{"namespaces/store_insert/namespace.dql", "post"}, {"namespaces/store_write/namespace.dql", "patch"},
		{"namespaces/store_access/namespace.dql", "get"},
		{"reports/reader/report.dql", "get"}, {"reports/writer/report.dql", "patch"},
		{"reports/get/report.dql", "get"},
		{"reports/publish_guard/report.dql", "get"},
		{"report_publications/get/publication.dql", "get"},
		{"reports/store_run_access/access.dql", "get"},
		{"reports/store_capabilities/capability.dql", "get"},
		{"report_versions/reader/version.dql", "get"}, {"report_versions/writer/version.dql", "patch"},
		{"report_versions/store_preview_definition/definition.dql", "get"},
		{"report_views/reader/view.dql", "get"},
		{"report_parameters/reader/parameter.dql", "get"}, {"report_parameters/writer/parameter.dql", "patch"},
		{"report_cube_configs/reader/config.dql", "get"}, {"report_cube_configs/writer/config.dql", "patch"},
		{"report_mcp_exposures/reader/exposure.dql", "get"}, {"report_mcp_exposures/writer/exposure.dql", "patch"},
		{"report_resource_files/reader/file.dql", "get"}, {"report_resource_files/writer/file.dql", "patch"},
		{"report_resource_files/store_skill_content/file.dql", "get"},
		{"report_resource_files/store_download/file.dql", "get"}, {"report_resource_files/store_download_budget/budget.dql", "get"}, {"report_resource_files/store_snapshot/file.dql", "get"},
		{"resource_namespaces/store_usage/usage.dql", "get"},
		{"resource_namespaces/store_presence/usage.dql", "get"},
		{"resource_namespace_claims/store_write/claim.dql", "patch"},
		{"authorization_grants/store_read/grant.dql", "get"},
		{"report_resource_folders/reader/folder.dql", "get"}, {"report_resource_folders/writer/folder.dql", "patch"},
		{"report_resource_folders/store_snapshot/folder.dql", "get"},
		{"report_skill_roots/reader/skill.dql", "get"}, {"report_skill_roots/writer/skill.dql", "patch"},
		{"report_skill_roots/store_validation/skill.dql", "get"},
		{"report_skill_roots/store_snapshot/skill.dql", "get"},
		{"runtime_generations/reader/generation.dql", "get"}, {"runtime_generations/writer/generation.dql", "patch"},
		{"runtime_generations/store_active/definition.dql", "get"}, {"runtime_generations/store_candidate/definition.dql", "get"},
		{"runtime_generations/store_status/generation.dql", "get"},
		{"runtime_generations/store_readers/reader.dql", "get"},
		{"runtime_generations/store_head/head.dql", "get"},
		{"runtime_generations/store_insert/generation.dql", "post"},
		{"runtime_generations/store_state/generation.dql", "patch"},
		{"runtime_generations/store_active_others/generation.dql", "get"},
		{"report_warmup_runs/reader/warmup_run.dql", "get"},
		{"report_warmup_runs/get/warmup_run.dql", "get"},
		{"report_warmup_runs/store_read/warmup_run.dql", "get"},
		{"report_warmup_runs/store_expired/warmup_run.dql", "get"},
		{"report_warmup_runs/store_write/warmup_run.dql", "patch"},
		{"report_publication_events/reader/event.dql", "get"},
		{"report_publication_events/list/event.dql", "get"},
		{"report_publication_events/store_list/event.dql", "get"},
		{"report_publication_events/store_owner/report.dql", "get"},
		{"report_publication_events/store_insert/event.dql", "post"},
		{"report_publications/store_status/publication.dql", "get"},
		{"report_publications/store_insert/publication.dql", "post"},
		{"report_publications/store_stage/publication.dql", "patch"},
		{"report_publications/store_activate/publication.dql", "patch"},
		{"report_publications/store_active_others/publication.dql", "get"},
		{"report_publications/store_repoint/publication.dql", "patch"},
		{"report_publications/store_runtime_catalog/publication.dql", "get"},
		{"bff_sessions/reader/session.dql", "get"}, {"bff_sessions/writer/session.dql", "patch"},
		{"bff_sessions/store_read/session.dql", "get"},
		{"authorization_predicates/store_read/authorization_predicate.dql", "get"},
		{"authorization_predicates/store_insert/authorization_predicate.dql", "post"},
		{"authorization_predicates/store_write/authorization_predicate.dql", "patch"},
		{"bff_sessions/store_expired/session.dql", "get"}, {"bff_sessions/store_write/session.dql", "patch"},
		{"report_publications/reader/publication.dql", "get"}, {"report_publications/writer/publication.dql", "patch"},
		{"report_acl/reader/acl.dql", "get"}, {"report_acl/writer/acl.dql", "patch"},
		{"report_acl/store_list/acl.dql", "get"}, {"report_acl/store_one/acl.dql", "get"},
		{"report_acl/store_write/acl.dql", "patch"},
	}
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "component_contract", "studio")
	module := datatest.NewGeneratedModule(t)
	types := datatest.StudioAuthorizationTypes(t)
	for _, item := range components {
		item := item
		t.Run(strings.TrimSuffix(item.path, ".dql"), func(t *testing.T) {
			serverOnly := strings.HasPrefix(item.path, "bff_sessions/store_") || strings.HasPrefix(item.path, "authorization_predicates/store_") || strings.HasPrefix(item.path, "authorization_grants/store_") || strings.HasPrefix(item.path, "connectors/store_") || strings.HasPrefix(item.path, "namespaces/store_") || strings.HasPrefix(item.path, "resource_namespaces/store_") || strings.HasPrefix(item.path, "resource_namespace_claims/store_") || strings.HasPrefix(item.path, "report_skill_roots/store_") || strings.HasPrefix(item.path, "report_resource_files/store_") || strings.HasPrefix(item.path, "report_resource_folders/store_") || strings.HasPrefix(item.path, "report_publication_events/store_") || strings.HasPrefix(item.path, "report_publications/store_") || strings.HasPrefix(item.path, "report_warmup_runs/store_") || strings.HasPrefix(item.path, "runtime_generations/store_") || strings.HasPrefix(item.path, "reports/store_") || strings.HasPrefix(item.path, "report_versions/store_") || strings.HasPrefix(item.path, "report_acl/store_")
			directory := filepath.Dir(item.path)
			payload, err := os.ReadFile(item.path)
			if err != nil {
				t.Fatal(err)
			}
			if !serverOnly {
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
			} else if strings.Contains(string(payload), "#import('github.com/viant/scy/auth/jwt')") || strings.Contains(string(payload), "$Jwt<") {
				t.Fatalf("server-only component %s must not depend on browser JWT", item.path)
			}
			if strings.Contains(string(payload), "WithPredicate(99, 'handler', 'authorization.") || strings.Contains(string(payload), "$predicate.FilterGroup(99") {
				t.Fatalf("component %s incorrectly couples JWT authentication to a row predicate", item.path)
			}
			if item.operation == "patch" && !serverOnly && item.path != "connectors/writer/connector.dql" && item.path != "namespaces/writer/namespace.dql" && item.path != "reports/writer/report.dql" {
				if !strings.Contains(string(payload), "lifecycle_type(") {
					t.Fatalf("writer %s is missing its parent authorization lifecycle", item.path)
				}
			}
			generatedInput, err := os.ReadFile(filepath.Join("..", "..", "studio", directory, "input.go"))
			if err != nil {
				t.Fatalf("read generated input for %s: %v", item.path, err)
			}
			if !serverOnly {
				for _, required := range []string{
					"*jwt.Claims",
					"in=Authorization,dataType=string,errorCode=401,required=true",
					`codec:"JwtClaim"`,
				} {
					if !strings.Contains(string(generatedInput), required) {
						t.Fatalf("generated input for %s is missing authorization contract %q", item.path, required)
					}
				}
			} else if strings.Contains(string(generatedInput), "*jwt.Claims") {
				t.Fatalf("server-only component %s exposes browser JWT", item.path)
			}
			if strings.Contains(string(generatedInput), `predicate:"handler,group=99,github.com/viant/datly-studio/studio/authorization.`) {
				t.Fatalf("generated input for %s incorrectly couples JWT authentication to a row predicate", item.path)
			}
			if item.operation == "patch" && !serverOnly && item.path != "connectors/writer/connector.dql" && item.path != "namespaces/writer/namespace.dql" && item.path != "reports/writer/report.dql" {
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
			generated, err := (transcribe.Generator{Operation: item.operation}).Generate(ctx, transcribe.GenerationRequest{Compiled: compiled, Destination: module.Root})
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
