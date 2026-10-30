package host

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly/application"
	"github.com/viant/datly/bootstrap"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/transcribe"
	identity "github.com/viant/forge/backend/resource"
	xhandler "github.com/viant/xdatly/handler"
)

type exactDependencyInput struct{}
type exactDependencyOutput struct{ Value string }
type exactDependencyHandler struct{}

func (exactDependencyHandler) Exec(_ context.Context, _ xhandler.Session, _ *exactDependencyInput, output *exactDependencyOutput) error {
	output.Value = "allowed"
	return nil
}

func exactDependency(t *testing.T) *registry.RegisteredComponent {
	t.Helper()
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: &spec.Component{Key: spec.Key{Kind: spec.KindComponent, Scope: "example.com/support", Name: "Context"}, Routes: []*spec.Route{{Method: http.MethodGet, Path: "/support-context", Internal: true}}}, InputType: reflect.TypeFor[exactDependencyInput](), OutputType: reflect.TypeFor[exactDependencyOutput]()})
	if err != nil {
		t.Fatal(err)
	}
	registered, err := artifact.Registration(registry.RegisteredComponent{Handler: custom.New[exactDependencyInput, exactDependencyOutput](exactDependencyHandler{})})
	if err != nil {
		t.Fatal(err)
	}
	return registered
}

func TestExactBuildDependencyRejectsPublicationAndClosesFailures(t *testing.T) {
	ctx := context.Background()
	failure := errors.New("dependency unavailable")
	for _, mode := range []string{"resolver", "public", "mcp", "key", "route", "fingerprint"} {
		t.Run(mode, func(t *testing.T) {
			closed, supportClosed := 0, 0
			dependency := exactDependency(t)
			source := exactDependency(t)
			source.Component.Key.Name = "Reader"
			source.Component.Routes[0].Path = "/reader"
			fingerprint := identity.ContentFingerprint([]byte("support"))
			var resolveErr error
			switch mode {
			case "resolver":
				resolveErr = failure
			case "public":
				dependency.Component.Routes[0].Internal = false
			case "mcp":
				dependency.Component.Routes[0].MCP = []*spec.MCPExposure{{Kind: spec.MCPExposureTool, Name: "support"}}
			case "key":
				dependency.Component.Key = source.Component.Key
			case "route":
				dependency.Component.Routes[0].Path = "/reader"
			case "fingerprint":
				fingerprint = ""
			}
			service := &Service{config: Config{ExactBuildDependencyResolver: func(context.Context, string, ComponentReference) (*ExactBuildDependencies, error) {
				return &ExactBuildDependencies{Components: []*registry.RegisteredComponent{dependency}, ContentFingerprint: fingerprint, Shutdown: func(context.Context) error { supportClosed++; return nil }}, resolveErr
			}}}
			build := &application.Build{Components: []*registry.RegisteredComponent{source}, Shutdown: func(context.Context) error { closed++; return nil }}
			_, err := service.addExactBuildDependencies(ctx, ComponentReference{}, build)
			if err == nil || closed != 1 || supportClosed != 1 || len(build.Components) != 1 {
				t.Fatalf("failure not closed: %v %d %d", err, closed, supportClosed)
			}
			if mode == "resolver" && !errors.Is(err, failure) {
				t.Fatal(err)
			}
		})
	}
}

func TestExactBuildDependencyIsFreshAndVersionScoped(t *testing.T) {
	dql := strings.Replace(unscopedRecordsDQL, "#define($_ = $Records", "#define($_ = $Support<?>(component/GET:/support-context).Required())\n#define($_ = $Records", 1)
	fixture, err := newScopedHost(t, map[string]string{"records": dql}, map[string]authz.Policy{"records": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}})
	if err != nil {
		t.Fatal(err)
	}
	service := fixture.service
	ctx := oauth.WithBearer(context.Background(), fixture.token(t, "alice", []authz.Entity{}))
	store, err := authz.NewStaticStore([]authz.Document{{Resource: authz.Resource{Kind: "component", ID: "records", Version: "1", Tenant: scopeTestTenant}, Revision: 1, Policies: map[string]authz.Policy{"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}, "execute": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	service.resourceAccess.Store = store

	if _, err = service.studio.ExecContext(ctx, "UPDATE component_versions SET state='validated' WHERE report_id='records' AND version_no=1"); err != nil {
		t.Fatal(err)
	}
	ref := ComponentReference{Kind: "dynamic", ID: "records", Revision: "1", Method: http.MethodGet, Route: "/records"}
	missingPin, err := service.ResolveComponentBinding(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	if result, executeErr := service.ExecuteComponentJSON(ctx, ref, missingPin, nil); executeErr == nil || len(result) != 0 {
		t.Fatal("missing native dependency executed")
	}

	calls, closed := 0, 0
	revision := "support-v1"
	service.config.NamespaceBuildResolver = func(context.Context, string) (NamespaceBuildExtension, error) {
		t.Fatal("namespace resolver called for exact reader")
		return nil, nil
	}
	service.config.ExactBuildDependencyResolver = func(callCtx context.Context, namespace string, selected ComponentReference) (*ExactBuildDependencies, error) {
		if callCtx != ctx || namespace != service.config.NamespaceID || selected != ref {
			t.Fatal("exact source/context changed")
		}
		calls++
		return &ExactBuildDependencies{Components: []*registry.RegisteredComponent{exactDependency(t)}, ContentFingerprint: identity.ContentFingerprint([]byte(revision)), Shutdown: func(context.Context) error { closed++; return nil }}, nil
	}

	if _, err := service.ResolveComponentBinding(context.Background(), ref); err == nil || calls != 0 {
		t.Fatal("unauthorized exact build invoked dependency resolver")
	}
	discoveries := 0
	service.compileRuntimeContract = func(callCtx context.Context, source *transcribe.Source) (*transcribe.RuntimeContract, error) {
		discoveries++
		return transcribe.NewCompiler().RuntimeContracts(callCtx, service.config.RootDir, source)
	}
	first, err := service.ResolveComponentBinding(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.ResolveComponentBinding(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	if discoveries != 0 {
		t.Fatalf("cached reader contract rediscovered: %d", discoveries)
	}
	if first != second || calls != 2 || closed != 2 {
		t.Fatalf("exact build reuse/lifecycle: %d %d", calls, closed)
	}

	manager, _, err := service.componentRuntime(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Shutdown(ctx)
	output, err := executeRuntimeJSON(ctx, service.authenticated(manager), http.MethodGet, "/records", nil)
	if err != nil || !strings.Contains(string(output), "alpha-101") {
		t.Fatalf("dependency-backed reader: %s %v", output, err)
	}
	if output, err := executeRuntimeJSON(ctx, service.authenticated(manager), http.MethodGet, "/support-context", nil); err == nil || len(output) != 0 {
		t.Fatal("internal dependency became externally callable")
	}
	revision = "support-v2"
	third, err := service.ResolveComponentBinding(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	if output, err := service.ExecuteComponentJSON(ctx, ref, first, nil); err == nil || len(output) != 0 {
		t.Fatal("stale support fingerprint executed")
	}
	if third.ContentFingerprint == first.ContentFingerprint || third.SchemaFingerprint != first.SchemaFingerprint {
		t.Fatal("support content missing from exact pin")
	}
	// Moving publication does not change the selected retained exact source.
	_, err = service.studio.ExecContext(ctx, `INSERT INTO component_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at)
 SELECT report_id,2,'validated',authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,'hash-v2',type_manifest_json,compile_status,datly_version,compiler_version,2,created_by,created_at FROM component_versions WHERE report_id='records' AND version_no=1`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.studio.ExecContext(ctx, "UPDATE component_publications SET active_version_no=2 WHERE report_id='records'"); err != nil {
		t.Fatal(err)
	}
	retained, err := service.ResolveComponentBinding(ctx, ref)
	if err != nil || retained != third {
		t.Fatalf("publication changed retained source: %v", err)
	}
	ref.Revision = "2"
	newer, binding, err := service.componentRuntimeWithAuthority(ctx, ref, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := newer.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	if binding.Revision != "2" || binding.ContentFingerprint == retained.ContentFingerprint {
		t.Fatal("exact dependency build lost selected version")
	}

}

func TestExactBuildDependencySuccessOwnsShutdown(t *testing.T) {
	ctx := context.Background()
	closed, supportClosed := 0, 0
	service := &Service{config: Config{ExactBuildDependencyResolver: func(context.Context, string, ComponentReference) (*ExactBuildDependencies, error) {
		return &ExactBuildDependencies{Components: []*registry.RegisteredComponent{exactDependency(t)}, ContentFingerprint: identity.ContentFingerprint([]byte("support")), Shutdown: func(context.Context) error { supportClosed++; return nil }}, nil
	}}}
	build := &application.Build{Shutdown: func(context.Context) error { closed++; return nil }}
	fingerprint, err := service.addExactBuildDependencies(ctx, ComponentReference{}, build)
	if err != nil || fingerprint == "" || closed != 0 || supportClosed != 0 || len(build.Components) != 1 {
		t.Fatalf("dependency ownership transferred incorrectly: %v", err)
	}
	if err := build.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	if closed != 1 || supportClosed != 1 {
		t.Fatalf("shutdown counts %d %d", closed, supportClosed)
	}
}
