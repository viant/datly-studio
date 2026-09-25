package dependencylink

import (
	"context"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/viant/datly/bootstrap"
	standaloneconfig "github.com/viant/datly/standalone/config"
)

var linkedComponentHolders = []struct {
	packagePath string
	holder      string
}{
	{"github.com/viant/datly-studio/studio/auth/reader", "ContextComponent"},
	{"github.com/viant/datly-studio/studio/bff_sessions/reader", "SessionComponent"},
	{"github.com/viant/datly-studio/studio/bff_sessions/writer", "SessionComponent"},
	{"github.com/viant/datly-studio/studio/connectors/reader", "ConnectorComponent"},
	{"github.com/viant/datly-studio/studio/connectors/get", "ConnectorComponent"},
	{"github.com/viant/datly-studio/studio/connectors/writer", "ConnectorComponent"},
	{"github.com/viant/datly-studio/studio/namespaces/reader", "NamespaceComponent"},
	{"github.com/viant/datly-studio/studio/namespaces/get", "NamespaceComponent"},
	{"github.com/viant/datly-studio/studio/namespaces/writer", "NamespaceComponent"},
	{"github.com/viant/datly-studio/studio/report_versions/reader", "VersionComponent"},
	{"github.com/viant/datly-studio/studio/report_versions/writer", "VersionComponent"},
	{"github.com/viant/datly-studio/studio/report_views/reader", "ViewComponent"},
	{"github.com/viant/datly-studio/studio/report_parameters/reader", "ParameterComponent"},
	{"github.com/viant/datly-studio/studio/report_parameters/writer", "ParameterComponent"},
	{"github.com/viant/datly-studio/studio/report_cube_configs/reader", "ConfigComponent"},
	{"github.com/viant/datly-studio/studio/report_cube_configs/writer", "ConfigComponent"},
	{"github.com/viant/datly-studio/studio/report_mcp_exposures/reader", "ExposureComponent"},
	{"github.com/viant/datly-studio/studio/report_mcp_exposures/writer", "ExposureComponent"},
	{"github.com/viant/datly-studio/studio/report_resource_files/reader", "FileComponent"},
	{"github.com/viant/datly-studio/studio/report_resource_files/writer", "FileComponent"},
	{"github.com/viant/datly-studio/studio/report_resource_folders/reader", "FolderComponent"},
	{"github.com/viant/datly-studio/studio/report_resource_folders/writer", "FolderComponent"},
	{"github.com/viant/datly-studio/studio/report_skill_roots/reader", "SkillComponent"},
	{"github.com/viant/datly-studio/studio/report_skill_roots/writer", "SkillComponent"},
	{"github.com/viant/datly-studio/studio/runtime_generations/reader", "GenerationComponent"},
	{"github.com/viant/datly-studio/studio/runtime_generations/writer", "GenerationComponent"},
	{"github.com/viant/datly-studio/studio/report_warmup_runs/reader", "WarmupRunComponent"},
	{"github.com/viant/datly-studio/studio/report_publications/reader", "PublicationComponent"},
	{"github.com/viant/datly-studio/studio/report_publications/get", "PublicationComponent"},
	{"github.com/viant/datly-studio/studio/report_publications/writer", "PublicationComponent"},
	{"github.com/viant/datly-studio/studio/report_acl/reader", "AclComponent"},
	{"github.com/viant/datly-studio/studio/report_acl/writer", "AclComponent"},
	{"github.com/viant/datly-studio/studio/reports/reader", "ReportComponent"},
	{"github.com/viant/datly-studio/studio/reports/get", "ReportComponent"},
	{"github.com/viant/datly-studio/studio/reports/writer", "ReportComponent"},
}

func TestBlankImportsExposeLinkedComponents(t *testing.T) {
	for _, item := range linkedComponentHolders {
		if holder := bootstrap.LinkedHolder(nil, item.packagePath, item.holder); holder == nil {
			t.Fatalf("%s.%s is absent from runtime typelinks", item.packagePath, item.holder)
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
	if _, ok, resolveErr := reflected.Types.Resolve("package", authorizationPackage+".ConnectorRead"); resolveErr != nil || !ok {
		t.Errorf("configured authorization package does not expose ConnectorRead: found=%t err=%v", ok, resolveErr)
	}
	if _, ok, resolveErr := reflected.Types.Resolve("package", predicatePackage+".ReportCatalogRead"); resolveErr != nil || !ok {
		t.Errorf("configured catalog predicate package does not expose ReportCatalogRead: found=%t err=%v", ok, resolveErr)
	}
	if _, ok, resolveErr := reflected.Types.Resolve("package", "github.com/viant/datly-studio/studio/auth/reader.Output"); resolveErr != nil || !ok {
		t.Errorf("configured auth package does not expose Output: found=%t err=%v", ok, resolveErr)
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
		if component == nil || component.LinkedInputType == nil || checked[component.PackagePath] {
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
	for _, name := range []string{"ConnectorRead", "ConnectorEdit", "NamespaceRead", "ReportRead", "ReportEdit", "ReportPublish", "ReportVersionRead", "ReportVersionEdit", "ReportViewRead", "ReportParameterRead", "ReportParameterEdit", "ReportCubeRead", "ReportCubeEdit", "ReportMCPRead", "ReportMCPEdit", "ReportResourceFileRead", "ReportResourceFileEdit", "ReportResourceFolderRead", "ReportResourceFolderEdit", "ReportSkillRead", "ReportSkillEdit", "PublicationRead", "PublicationEdit", "ACLRead", "ACLEdit", "RuntimeRead", "RuntimeEdit"} {
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
