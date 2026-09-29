package dependencylink

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/viant/datly/bootstrap"
	"github.com/viant/datly/spec"
	standaloneconfig "github.com/viant/datly/standalone/config"
)

func TestDependencyLinkContainsOnlyBlankImports(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "link.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Decls) != 1 {
		t.Fatalf("dependency link has %d declarations, want one import block", len(file.Decls))
	}
	declaration, ok := file.Decls[0].(*ast.GenDecl)
	if !ok || declaration.Tok != token.IMPORT {
		t.Fatal("dependency link contains a non-import declaration")
	}
	for _, item := range declaration.Specs {
		linked, ok := item.(*ast.ImportSpec)
		if !ok || linked.Name == nil || linked.Name.Name != "_" {
			t.Fatalf("dependency link has a non-blank import: %#v", item)
		}
	}
}

var linkedComponentHolders = []struct {
	packagePath string
	holder      string
}{
	{"github.com/viant/datly-studio/studio/auth/reader", "ContextComponent"},
	{"github.com/viant/datly-studio/studio/authorization_predicates/types", "Component"},
	{"github.com/viant/datly-studio/studio/authorization_predicates/get", "Component"},
	{"github.com/viant/datly-studio/studio/authorization_predicates/create", "Component"},
	{"github.com/viant/datly-studio/studio/authorization_predicates/update", "Component"},
	{"github.com/viant/datly-studio/studio/authorization_predicates/delete", "Component"},
	{"github.com/viant/datly-studio/studio/authorization_predicates/list", "Component"},
	{"github.com/viant/datly-studio/studio/authorization_predicates/store_read", "AuthorizationPredicateComponent"},
	{"github.com/viant/datly-studio/studio/authorization_predicates/store_insert", "AuthorizationPredicateComponent"},
	{"github.com/viant/datly-studio/studio/authorization_predicates/store_write", "AuthorizationPredicateComponent"},
	{"github.com/viant/datly-studio/studio/bff_sessions/reader", "SessionComponent"},
	{"github.com/viant/datly-studio/studio/bff_sessions/writer", "SessionComponent"},
	{"github.com/viant/datly-studio/studio/connectors/reader", "ConnectorComponent"},
	{"github.com/viant/datly-studio/studio/connectors/get", "ConnectorComponent"},
	{"github.com/viant/datly-studio/studio/connectors/disable", "Component"},
	{"github.com/viant/datly-studio/studio/connectors/delete", "Component"},
	{"github.com/viant/datly-studio/studio/connectors/activate", "Component"},
	{"github.com/viant/datly-studio/studio/connectors/store_access", "ConnectorComponent"},
	{"github.com/viant/datly-studio/studio/connectors/store_catalog", "ConnectorComponent"},
	{"github.com/viant/datly-studio/studio/connectors/store_config", "ConnectorComponent"},
	{"github.com/viant/datly-studio/studio/connectors/store_status", "ConnectorComponent"},
	{"github.com/viant/datly-studio/studio/connectors/store_usage", "UsageComponent"},
	{"github.com/viant/datly-studio/studio/connectors/update", "Component"},
	{"github.com/viant/datly-studio/studio/connectors/create", "ConnectorComponent"},
	{"github.com/viant/datly-studio/studio/connectors/schemas", "Component"},
	{"github.com/viant/datly-studio/studio/connectors/tables", "Component"},
	{"github.com/viant/datly-studio/studio/connectors/table", "Component"},
	{"github.com/viant/datly-studio/studio/connectors/test", "Component"},
	{"github.com/viant/datly-studio/studio/connectors/test_sql", "Component"},
	{"github.com/viant/datly-studio/studio/connectors/writer", "ConnectorComponent"},
	{"github.com/viant/datly-studio/studio/namespaces/reader", "NamespaceComponent"},
	{"github.com/viant/datly-studio/studio/namespaces/get", "NamespaceComponent"},
	{"github.com/viant/datly-studio/studio/namespaces/store_read", "NamespaceComponent"},
	{"github.com/viant/datly-studio/studio/namespaces/store_usage", "UsageComponent"},
	{"github.com/viant/datly-studio/studio/namespaces/store_write", "NamespaceComponent"},
	{"github.com/viant/datly-studio/studio/namespaces/update", "Component"},
	{"github.com/viant/datly-studio/studio/namespaces/create", "NamespaceComponent"},
	{"github.com/viant/datly-studio/studio/namespaces/delete", "Component"},
	{"github.com/viant/datly-studio/studio/namespaces/writer", "NamespaceComponent"},
	{"github.com/viant/datly-studio/studio/report_versions/reader", "VersionComponent"},
	{"github.com/viant/datly-studio/studio/report_versions/create", "Component"},
	{"github.com/viant/datly-studio/studio/report_versions/clone", "Component"},
	{"github.com/viant/datly-studio/studio/report_versions/load_dql", "Component"},
	{"github.com/viant/datly-studio/studio/report_versions/load_archive", "Component"},
	{"github.com/viant/datly-studio/studio/report_versions/inspect", "Component"},
	{"github.com/viant/datly-studio/studio/report_versions/apply", "Component"},
	{"github.com/viant/datly-studio/studio/report_versions/builder", "Component"},
	{"github.com/viant/datly-studio/studio/report_versions/validate", "Component"},
	{"github.com/viant/datly-studio/studio/report_versions/test_view", "Component"},
	{"github.com/viant/datly-studio/studio/report_versions/test_relation", "Component"},
	{"github.com/viant/datly-studio/studio/report_versions/test_compose", "Component"},
	{"github.com/viant/datly-studio/studio/report_versions/warmup", "Component"},
	{"github.com/viant/datly-studio/studio/report_versions/store_head", "HeadComponent"},
	{"github.com/viant/datly-studio/studio/report_versions/store_insert", "VersionComponent"},
	{"github.com/viant/datly-studio/studio/report_versions/store_import", "VersionComponent"},
	{"github.com/viant/datly-studio/studio/report_versions/store_catalog", "VersionComponent"},
	{"github.com/viant/datly-studio/studio/report_versions/store_edit", "VersionComponent"},
	{"github.com/viant/datly-studio/studio/report_versions/store_touch", "VersionComponent"},
	{"github.com/viant/datly-studio/studio/report_versions/store_validation", "VersionComponent"},
	{"github.com/viant/datly-studio/studio/report_versions/get", "VersionComponent"},
	{"github.com/viant/datly-studio/studio/report_versions/export_dql", "VersionComponent"},
	{"github.com/viant/datly-studio/studio/report_versions/descriptor", "VersionComponent"},
	{"github.com/viant/datly-studio/studio/report_versions/download", "Component"},
	{"github.com/viant/datly-studio/studio/report_versions/list", "VersionComponent"},
	{"github.com/viant/datly-studio/studio/report_versions/writer", "VersionComponent"},
	{"github.com/viant/datly-studio/studio/report_views/reader", "ViewComponent"},
	{"github.com/viant/datly-studio/studio/report_parameters/reader", "ParameterComponent"},
	{"github.com/viant/datly-studio/studio/report_parameters/writer", "ParameterComponent"},
	{"github.com/viant/datly-studio/studio/report_cube_configs/reader", "ConfigComponent"},
	{"github.com/viant/datly-studio/studio/report_cube_configs/writer", "ConfigComponent"},
	{"github.com/viant/datly-studio/studio/report_mcp_exposures/reader", "ExposureComponent"},
	{"github.com/viant/datly-studio/studio/report_mcp_exposures/writer", "ExposureComponent"},
	{"github.com/viant/datly-studio/studio/report_resource_files/reader", "FileComponent"},
	{"github.com/viant/datly-studio/studio/report_resource_files/store_download", "FileComponent"},
	{"github.com/viant/datly-studio/studio/report_resource_files/store_download_budget", "BudgetComponent"},
	{"github.com/viant/datly-studio/studio/report_resource_files/store_snapshot", "FileComponent"},
	{"github.com/viant/datly-studio/studio/report_resource_files/store_write", "FileComponent"},
	{"github.com/viant/datly-studio/studio/report_resource_files/writer", "FileComponent"},
	{"github.com/viant/datly-studio/studio/report_resource_folders/reader", "FolderComponent"},
	{"github.com/viant/datly-studio/studio/report_resource_folders/store_snapshot", "FolderComponent"},
	{"github.com/viant/datly-studio/studio/report_resource_folders/store_write", "FolderComponent"},
	{"github.com/viant/datly-studio/studio/report_resource_folders/writer", "FolderComponent"},
	{"github.com/viant/datly-studio/studio/report_skill_roots/reader", "SkillComponent"},
	{"github.com/viant/datly-studio/studio/report_skill_roots/store_snapshot", "SkillComponent"},
	{"github.com/viant/datly-studio/studio/report_skill_roots/store_write", "SkillComponent"},
	{"github.com/viant/datly-studio/studio/report_skill_roots/writer", "SkillComponent"},
	{"github.com/viant/datly-studio/studio/runtime_generations/reader", "GenerationComponent"},
	{"github.com/viant/datly-studio/studio/runtime_generations/writer", "GenerationComponent"},
	{"github.com/viant/datly-studio/studio/runtime/status", "Component"},
	{"github.com/viant/datly-studio/studio/report_warmup_runs/reader", "WarmupRunComponent"},
	{"github.com/viant/datly-studio/studio/report_warmup_runs/get", "WarmupRunComponent"},
	{"github.com/viant/datly-studio/studio/report_warmup_runs/list", "Component"},
	{"github.com/viant/datly-studio/studio/report_warmup_runs/store_expired", "WarmupRunComponent"},
	{"github.com/viant/datly-studio/studio/report_warmup_runs/store_read", "WarmupRunComponent"},
	{"github.com/viant/datly-studio/studio/report_warmup_runs/store_write", "WarmupRunComponent"},
	{"github.com/viant/datly-studio/studio/report_publications/reader", "PublicationComponent"},
	{"github.com/viant/datly-studio/studio/report_publications/get", "PublicationComponent"},
	{"github.com/viant/datly-studio/studio/report_publications/mutate", "PublishComponent"},
	{"github.com/viant/datly-studio/studio/report_publications/mutate", "RollbackComponent"},
	{"github.com/viant/datly-studio/studio/report_publications/mutate", "UnpublishComponent"},
	{"github.com/viant/datly-studio/studio/report_publications/writer", "PublicationComponent"},
	{"github.com/viant/datly-studio/studio/report_publication_events/list", "EventComponent"},
	{"github.com/viant/datly-studio/studio/report_acl/reader", "AclComponent"},
	{"github.com/viant/datly-studio/studio/report_acl/delete", "Component"},
	{"github.com/viant/datly-studio/studio/report_acl/upsert", "Component"},
	{"github.com/viant/datly-studio/studio/report_acl/store_write", "AclComponent"},
	{"github.com/viant/datly-studio/studio/report_acl/writer", "AclComponent"},
	{"github.com/viant/datly-studio/studio/reports/reader", "ReportComponent"},
	{"github.com/viant/datly-studio/studio/reports/get", "ReportComponent"},
	{"github.com/viant/datly-studio/studio/reports/create", "Component"},
	{"github.com/viant/datly-studio/studio/reports/edit_guard", "ReportComponent"},
	{"github.com/viant/datly-studio/studio/reports/store_insert", "ReportComponent"},
	{"github.com/viant/datly-studio/studio/reports/store_catalog", "ReportComponent"},
	{"github.com/viant/datly-studio/studio/reports/store_config", "ReportComponent"},
	{"github.com/viant/datly-studio/studio/reports/store_draft_pointer", "ReportComponent"},
	{"github.com/viant/datly-studio/studio/reports/store_global_access", "ReportComponent"},
	{"github.com/viant/datly-studio/studio/reports/store_capabilities", "CapabilityComponent"},
	{"github.com/viant/datly-studio/studio/reports/update", "Component"},
	{"github.com/viant/datly-studio/studio/reports/publish_guard", "ReportComponent"},
	{"github.com/viant/datly-studio/studio/reports/store_run_access", "AccessComponent"},
	{"github.com/viant/datly-studio/studio/preview/execute", "Component"},
	{"github.com/viant/datly-studio/studio/reports/writer", "ReportComponent"},
	{"github.com/viant/datly-studio/studio/resources/get", "Component"},
	{"github.com/viant/datly-studio/studio/resources/mutate", "FileUpsertComponent"},
	{"github.com/viant/datly-studio/studio/resources/mutate", "FileDeleteComponent"},
	{"github.com/viant/datly-studio/studio/resources/mutate", "FolderUpsertComponent"},
	{"github.com/viant/datly-studio/studio/resources/mutate", "FolderDeleteComponent"},
	{"github.com/viant/datly-studio/studio/resources/mutate", "SkillUpsertComponent"},
	{"github.com/viant/datly-studio/studio/resources/mutate", "SkillDeleteComponent"},
	{"github.com/viant/datly-studio/studio/resource_namespace_claims/store_write", "ClaimComponent"},
	{"github.com/viant/datly-studio/studio/resource_namespaces/store_presence", "UsageComponent"},
	{"github.com/viant/datly-studio/studio/resource_namespaces/store_usage", "UsageComponent"},
	{"github.com/viant/authz/datly/policy/reader", "PolicyComponent"},
	{"github.com/viant/authz/datly/policy/writer", "PolicyComponent"},
	{"github.com/viant/datly-studio/studio/resource_policy/access", "GetComponent"},
	{"github.com/viant/datly-studio/studio/resource_policy/access", "ContextComponent"},
	{"github.com/viant/datly-studio/studio/resource_policy/access", "ReplaceComponent"},
	{"github.com/viant/datly-studio/studio/resource_policy/catalog", "Component"},
}

func TestBlankImportsExposeLinkedComponents(t *testing.T) {
	for _, item := range linkedComponentHolders {
		if holder := bootstrap.LinkedHolder(nil, item.packagePath, item.holder); holder == nil {
			t.Fatalf("%s.%s is absent from runtime typelinks", item.packagePath, item.holder)
		}
	}
}

func TestDownloadUsesLinkedNativeHandlerAndPrivateResourceReader(t *testing.T) {
	reflected, err := bootstrap.ReflectPackages([]string{
		"github.com/viant/datly-studio/studio/report_versions/download",
		"github.com/viant/datly-studio/studio/report_resource_files/store_download",
	})
	if err != nil {
		t.Fatal(err)
	}
	var handlerFound, privateReaderFound bool
	for _, component := range reflected.Components {
		switch component.PackagePath {
		case "github.com/viant/datly-studio/studio/report_versions/download":
			if component.Tag.Internal || component.Tag.Handler != "NewDownload" || component.LinkedHandler == nil {
				t.Fatalf("native download handler is not linked: %+v", component.Tag)
			}
			if handler, handlerErr := component.LinkedHandler(); handlerErr != nil || handler == nil {
				t.Fatalf("native download handler factory: %v", handlerErr)
			}
			handlerFound = true
		case "github.com/viant/datly-studio/studio/report_resource_files/store_download":
			if !component.Tag.Internal || len(component.Tag.MCP) != 0 {
				t.Fatalf("resource child reader must be internal-only: %+v", component.Tag)
			}
			privateReaderFound = true
		}
	}
	if !handlerFound || !privateReaderFound {
		t.Fatalf("linked native download=%t internal resource reader=%t", handlerFound, privateReaderFound)
	}
}

func TestNamespaceCreateSelectsBuiltInMutationWriter(t *testing.T) {
	reflected, err := bootstrap.ReflectPackages([]string{"github.com/viant/datly-studio/studio/namespaces/create"})
	if err != nil {
		t.Fatal(err)
	}
	if len(reflected.Components) != 1 {
		t.Fatalf("native namespace create components=%d, want 1", len(reflected.Components))
	}
	component := reflected.Components[0]
	if component.Tag.Handler != "" || component.Tag.Settings.Mutation != "post" || component.Tag.Internal || component.Tag.Path != "/v1/studio/sdk/namespaces.create" {
		t.Fatalf("native namespace writer metadata=%+v", component.Tag)
	}
}

func TestConnectorCreateSelectsBuiltInMutationWriter(t *testing.T) {
	reflected, err := bootstrap.ReflectPackages([]string{"github.com/viant/datly-studio/studio/connectors/create"})
	if err != nil {
		t.Fatal(err)
	}
	if len(reflected.Components) != 1 {
		t.Fatalf("native connector create components=%d, want 1", len(reflected.Components))
	}
	component := reflected.Components[0]
	if component.Tag.Handler != "" || component.Tag.Settings.Mutation != "post" || component.Tag.Internal || component.Tag.Path != "/v1/studio/sdk/connectors.create" {
		t.Fatalf("native connector writer metadata=%+v", component.Tag)
	}
}

func TestResourceSnapshotUsesLinkedNativeHandlerAndPrivateReaders(t *testing.T) {
	packages := []string{
		"github.com/viant/datly-studio/studio/resources/get",
		"github.com/viant/datly-studio/studio/report_resource_files/store_snapshot",
		"github.com/viant/datly-studio/studio/report_resource_folders/store_snapshot",
		"github.com/viant/datly-studio/studio/report_skill_roots/store_snapshot",
	}
	reflected, err := bootstrap.ReflectPackages(packages)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, component := range reflected.Components {
		seen[component.PackagePath] = true
		if component.PackagePath == packages[0] {
			if component.Tag.Internal || component.Tag.Handler != "NewResourceSnapshot" || component.LinkedHandler == nil {
				t.Fatalf("native snapshot handler is not linked: %+v", component.Tag)
			}
			if handler, handlerErr := component.LinkedHandler(); handlerErr != nil || handler == nil {
				t.Fatalf("native snapshot handler factory: %v", handlerErr)
			}
			continue
		}
		if !component.Tag.Internal || len(component.Tag.MCP) != 0 {
			t.Fatalf("snapshot child reader must be internal-only: %+v", component.Tag)
		}
	}
	for _, path := range packages {
		if !seen[path] {
			t.Fatalf("snapshot package %s is not linked", path)
		}
	}
}

func TestDatlyConfigurationSelectsEveryLinkedComponentPackage(t *testing.T) {
	configurationPath, err := filepath.Abs(filepath.Join("..", "..", "datly.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	configuration, err := (standaloneconfig.Loader{}).Load(context.Background(), configurationPath)
	if err != nil {
		t.Fatalf("load datly.yaml: %v", err)
	}
	if configuration.GoBootstrap == nil || !configuration.GoBootstrap.EagerComponents {
		t.Fatal("datly.yaml must eagerly bootstrap linked Studio components")
	}
	reflected, err := bootstrap.ReflectPackages(configuration.GoBootstrap.Packages)
	if err != nil {
		t.Fatalf("reflect configured packages: %v", err)
	}
	configuredRoutes := make(map[string]int, len(reflected.Components))
	for _, component := range reflected.Components {
		configuredRoutes[component.PackagePath]++
	}
	wantPackages := make([]string, 0, len(linkedComponentHolders)+1)
	for _, item := range linkedComponentHolders {
		wantPackages = append(wantPackages, item.packagePath)
		if configuredRoutes[item.packagePath] == 0 {
			t.Errorf("configured package %s exposes no component routes", item.packagePath)
		}
	}
	authorizationPackage := "github.com/viant/datly-studio/studio/authorization"
	wantPackages = append(wantPackages, authorizationPackage)
	predicatePackage := "github.com/viant/datly-studio/studio/reports/catalogpredicate"
	wantPackages = append(wantPackages, predicatePackage)
	globalPredicatePackage := "github.com/viant/datly-studio/studio/reports/globalpredicate"
	wantPackages = append(wantPackages, globalPredicatePackage)
	connectorAccessPackage := "github.com/viant/datly-studio/studio/connectors/accesspredicate"
	wantPackages = append(wantPackages, connectorAccessPackage)
	if _, ok, resolveErr := reflected.Types.Resolve("package", authorizationPackage+".ConnectorRead"); resolveErr != nil || !ok {
		t.Errorf("configured authorization package does not expose ConnectorRead: found=%t err=%v", ok, resolveErr)
	}
	if _, ok, resolveErr := reflected.Types.Resolve("package", predicatePackage+".ReportCatalogRead"); resolveErr != nil || !ok {
		t.Errorf("configured catalog predicate package does not expose ReportCatalogRead: found=%t err=%v", ok, resolveErr)
	}
	if _, ok, resolveErr := reflected.Types.Resolve("package", globalPredicatePackage+".GlobalPublish"); resolveErr != nil || !ok {
		t.Errorf("configured global predicate package does not expose GlobalPublish: found=%t err=%v", ok, resolveErr)
	}
	if _, ok, resolveErr := reflected.Types.Resolve("package", connectorAccessPackage+".ConnectorAccess"); resolveErr != nil || !ok {
		t.Errorf("configured connector access package does not expose ConnectorAccess: found=%t err=%v", ok, resolveErr)
	}
	if _, ok, resolveErr := reflected.Types.Resolve("package", "github.com/viant/datly-studio/studio/auth/reader.Output"); resolveErr != nil || !ok {
		t.Errorf("configured auth package does not expose Output: found=%t err=%v", ok, resolveErr)
	}
	uniquePackages := make(map[string]bool, len(wantPackages))
	for _, path := range wantPackages {
		uniquePackages[path] = true
	}
	wantPackages = wantPackages[:0]
	for path := range uniquePackages {
		wantPackages = append(wantPackages, path)
	}
	sort.Strings(wantPackages)
	gotPackages := append([]string(nil), configuration.GoBootstrap.Packages...)
	sort.Strings(gotPackages)
	if len(gotPackages) != len(wantPackages) {
		t.Fatalf("datly.yaml package count=%d, want %d\ngot:  %v\nwant: %v", len(gotPackages), len(wantPackages), gotPackages, wantPackages)
	}
	for index := range wantPackages {
		if gotPackages[index] != wantPackages[index] {
			t.Fatalf("datly.yaml packages differ\ngot:  %v\nwant: %v", gotPackages, wantPackages)
		}
	}
}

func TestMCPDeclarationsMatchSelectedComponentMetadata(t *testing.T) {
	configurationPath, err := filepath.Abs(filepath.Join("..", "..", "datly.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	configuration, err := (standaloneconfig.Loader{}).Load(context.Background(), configurationPath)
	if err != nil {
		t.Fatal(err)
	}
	reflected, err := bootstrap.ReflectPackages(configuration.GoBootstrap.Packages)
	if err != nil {
		t.Fatal(err)
	}
	selected := map[string]bool{}
	for _, name := range configuration.GoBootstrap.Packages {
		selected[name] = true
	}
	linked := map[string]map[string]bool{}
	seen := map[string]string{}
	for _, component := range reflected.Components {
		for _, exposure := range component.Tag.MCP {
			if exposure == nil || exposure.Kind != spec.MCPExposureTool {
				continue
			}
			if prior := seen[exposure.Name]; prior != "" && prior != component.PackagePath {
				t.Errorf("MCP tool %q is declared by both %s and %s", exposure.Name, prior, component.PackagePath)
			}
			seen[exposure.Name] = component.PackagePath
			if linked[component.PackagePath] == nil {
				linked[component.PackagePath] = map[string]bool{}
			}
			linked[component.PackagePath][exposure.Name] = true
		}
	}
	// These old catalog readers are intentionally absent until their SDK wire
	// shapes and policy projections match the selected native endpoints.
	unselected := map[string]bool{
		"github.com/viant/datly-studio/studio/authorization_predicates/reader":  true,
		"github.com/viant/datly-studio/studio/report_publication_events/reader": true,
	}
	packageRE := regexp.MustCompile(`#package\('([^']+)'\)`)
	mcpRE := regexp.MustCompile(`\$mcp\('([^']+)'`)
	root := filepath.Join("..", "..", "dql", "studio")
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".dql" {
			return nil
		}
		payload, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		matches := mcpRE.FindAllStringSubmatch(string(payload), -1)
		if len(matches) == 0 {
			return nil
		}
		packageMatch := packageRE.FindStringSubmatch(string(payload))
		if len(packageMatch) != 2 {
			t.Errorf("MCP source %s lacks #package", path)
			return nil
		}
		packagePath := "github.com/viant/datly-studio/" + packageMatch[1]
		if !selected[packagePath] {
			if !unselected[packagePath] {
				t.Errorf("MCP source %s is absent from datly.yaml", packagePath)
			}
			return nil
		}
		for _, match := range matches {
			if !linked[packagePath][match[1]] {
				t.Errorf("MCP source %s declares %q without matching linked route metadata", packagePath, match[1])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestConfiguredComponentsRequireAuthenticationAndAuthorization(t *testing.T) {
	configurationPath, err := filepath.Abs(filepath.Join("..", "..", "datly.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	configuration, err := (standaloneconfig.Loader{}).Load(context.Background(), configurationPath)
	if err != nil {
		t.Fatal(err)
	}
	reflected, err := bootstrap.ReflectPackages(configuration.GoBootstrap.Packages)
	if err != nil {
		t.Fatal(err)
	}
	checked := map[string]bool{}
	for _, component := range reflected.Components {
		if component == nil || component.LinkedInputType == nil {
			continue
		}
		if component.Tag.Internal {
			if len(component.Tag.MCP) != 0 {
				t.Errorf("%s internal component declares MCP exposure", component.PackagePath)
			}
			continue
		}
		if checked[component.PackagePath] {
			continue
		}
		checked[component.PackagePath] = true
		input := component.LinkedInputType
		for input.Kind() == reflect.Pointer {
			input = input.Elem()
		}
		jwt, ok := input.FieldByName("Jwt")
		if !ok || !strings.Contains(jwt.Tag.Get("parameter"), "in=Authorization") || !strings.Contains(jwt.Tag.Get("parameter"), "required=true") || jwt.Tag.Get("codec") != "JwtClaim" {
			t.Errorf("%s input does not require verified JWT claims", component.PackagePath)
		}
		if component.PackagePath == "github.com/viant/datly-studio/studio/auth/reader" {
			continue
		}
		if strings.HasSuffix(component.PackagePath, "/reader") {
			authorized := false
			for index := 0; index < input.NumField(); index++ {
				field := input.Field(index)
				authorized = authorized || strings.Contains(string(field.Tag), `predicate:"handler`) || strings.Contains(field.Tag.Get("parameter"), "kind=component")
			}
			if !authorized {
				t.Errorf("%s reader has authentication but no authorization dependency/predicate", component.PackagePath)
			}
			continue
		}
		if strings.HasSuffix(component.PackagePath, "/writer") {
			_, hasInit := reflect.PointerTo(input).MethodByName("Init")
			authorized := hasInit
			for index := 0; index < input.NumField() && !authorized; index++ {
				authorized = strings.Contains(input.Field(index).Tag.Get("view"), "entityHooks=")
			}
			if !authorized {
				t.Errorf("%s writer has authentication but no authorization/lifecycle hook", component.PackagePath)
			}
		}
	}
}

func TestBlankImportsExposeAuthorizationHandlerTypes(t *testing.T) {
	for _, name := range []string{"ConnectorRead", "ConnectorEdit", "NamespaceRead", "ReportRead", "ReportEdit", "ReportPublish", "ReportVersionRead", "ReportVersionMetadataRead", "ReportVersionEdit", "ReportViewRead", "ReportParameterRead", "ReportParameterEdit", "ReportCubeRead", "ReportCubeEdit", "ReportMCPRead", "ReportMCPEdit", "ReportResourceFileRead", "ReportResourceFileEdit", "ReportResourceFolderRead", "ReportResourceFolderEdit", "ReportSkillRead", "ReportSkillEdit", "PublicationRead", "PublicationEdit", "ACLRead", "ACLEdit", "RuntimeRead", "RuntimeEdit", "WarmupRead"} {
		if bootstrap.LinkedHolder(nil, "github.com/viant/datly-studio/studio/authorization", name) == nil {
			t.Fatalf("authorization handler %s is absent from runtime typelinks", name)
		}
	}
}

func TestReflectionBootstrapDiscoversComponentsAndOrdinaryTypes(t *testing.T) {
	discovered, err := bootstrap.ReflectPackages([]string{
		"github.com/viant/datly-studio/studio/authorization",
		"github.com/viant/datly-studio/studio/connectors/reader",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(discovered.Components) != 1 {
		t.Fatalf("expected one native connector reader route, got %d", len(discovered.Components))
	}
	if _, ok, err := discovered.Types.Resolve("package", "github.com/viant/datly-studio/studio/authorization.ConnectorRead"); err != nil || !ok {
		t.Fatalf("ConnectorRead reflection discovery: found=%v err=%v", ok, err)
	}
}
