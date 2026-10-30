package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/datly/application"
	"github.com/viant/datly/bootstrap"
	dexec "github.com/viant/datly/exec"
	mcpserver "github.com/viant/datly/mcp/server"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/typecatalog"
	xhandler "github.com/viant/xdatly/handler"
)

func TestNamespaceBuildExtensionSelectionAndFailureCleanup(t *testing.T) {
	calls, closed := 0, 0
	failure := errors.New("composition failed")
	service := &Service{config: Config{NamespaceBuildExtensions: map[string]NamespaceBuildExtension{
		"selected": func(context.Context, *application.Build) error { calls++; return failure },
	}}}
	for _, namespace := range []string{"", "other"} {
		service.config.NamespaceID = namespace
		build := &application.Build{}
		actual, err := service.extendNamespaceBuild(context.Background(), build)
		if err != nil || actual != build || calls != 0 {
			t.Fatal("unselected namespace was extended")
		}
	}
	service.config.NamespaceID = "selected"
	build, err := service.extendNamespaceBuild(context.Background(), &application.Build{Shutdown: func(context.Context) error { closed++; return nil }})
	if build != nil || !errors.Is(err, failure) || calls != 1 || closed != 1 {
		t.Fatalf("failed build lifecycle: %v %v %d %d", build, err, calls, closed)
	}
}

type composingInput struct{}
type composingOutput struct {
	Data string `json:"data"`
}
type composingHandler struct{ target dexec.ComponentTarget }

func (h composingHandler) Exec(ctx context.Context, session xhandler.Session, _ *composingInput, output *composingOutput) error {
	value, found, err := session.Binder().Lookup(ctx, dexec.ComponentInvokerKey)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("native component invoker missing")
	}
	result, err := value.(dexec.ComponentInvoker).InvokeComponent(ctx, dexec.ComponentRequest{Target: h.target})
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(result)
	output.Data = string(encoded)
	return err
}

// This tests native runtime composition and dispatch, not transport authorization.
// Product entry-point policy must be supplied by the embedding application.
func TestNamespaceCustomHandlerInvokesDynamicReaderInMemory(t *testing.T) {
	fixture, err := newScopedHost(t, map[string]string{"records": unscopedRecordsDQL, "foreign": strings.ReplaceAll(strings.ReplaceAll(unscopedRecordsDQL, "runtime/records", "runtime/foreign"), "'/records'", "'/foreign'")}, map[string]authz.Policy{"records": {Mode: "public"}})
	if err != nil {
		t.Fatal(err)
	}
	service := fixture.service
	ctx := context.Background()
	namespaceID := namespaceaccess.ID("owner", "selected")
	if _, err = service.studio.ExecContext(ctx, `INSERT INTO namespaces(namespace_id,owner_id,name,title,status,visibility,mcp_enabled,etag,created_at,updated_at) VALUES(?,'owner','selected','Selected','active','public',TRUE,1,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, namespaceID); err != nil {
		t.Fatal(err)
	}
	if _, err = service.studio.ExecContext(ctx, `UPDATE components SET namespace='selected' WHERE id='records'`); err != nil {
		t.Fatal(err)
	}
	service.config.NamespaceID = namespaceID
	manager, err := application.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Shutdown(ctx)
	parent := dexec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: "example.com/custom", Name: "Run"}, Route: spec.RouteRef{Method: http.MethodGet, Path: "/run"}}
	extend := func(ctx context.Context, build *application.Build) error {
		var target dexec.ComponentTarget
		for _, item := range build.Components {
			for _, route := range item.Component.Routes {
				if route.Path == "/records" {
					target = dexec.ComponentTarget{Component: item.Component.Key, Route: spec.RouteRef{Method: route.Method, Path: route.Path}}
				}
			}
		}
		if target.Component.Name == "" {
			return fmt.Errorf("published reader missing")
		}
		artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: &spec.Component{Key: parent.Component, Routes: []*spec.Route{{Method: parent.Route.Method, Path: parent.Route.Path}}}, InputType: reflect.TypeFor[composingInput](), OutputType: reflect.TypeFor[composingOutput]()})
		if err != nil {
			return err
		}
		registered, err := artifact.Registration(registry.RegisteredComponent{Handler: custom.New[composingInput, composingOutput](composingHandler{target: target})})
		if err != nil {
			return err
		}
		build.Components = append(build.Components, registered)
		return nil
	}
	service.config.NamespaceBuildResolver = func(_ context.Context, id string) (NamespaceBuildExtension, error) {
		if id != namespaceID {
			return nil, nil
		}
		return extend, nil
	}

	err = manager.Reload(ctx, application.Request{Revision: 1, Compile: func(ctx context.Context, seed *typecatalog.Catalog) (*application.Build, error) {
		return service.compile(ctx, seed, nil)
	}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := manager.InvokeComponent(ctx, dexec.ComponentRequest{Target: parent})
	if err != nil {
		t.Fatal(err)
	}
	output := result.(*composingOutput)
	if !strings.Contains(output.Data, "alpha-101") || !strings.Contains(output.Data, "delta-104") {
		t.Fatalf("dynamic reader output missing: %s", output.Data)
	}
	// An admitted operation retains its reader generation while new calls see
	// the newly published revision, without rebuilding the handler binary.
	pinned, _, err := manager.PinOwned(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer mcpserver.Release(pinned)
	revised := strings.Replace(unscopedRecordsDQL, "t.name", "'revision-two' AS name", 1)
	_, err = service.studio.ExecContext(ctx, `INSERT INTO component_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at)
 SELECT report_id,2,'published',authoring_mode,?,?,component_spec_json,spec_format_version,'hash-v2',type_manifest_json,compile_status,datly_version,compiler_version,2,created_by,created_at FROM component_versions WHERE report_id='records' AND version_no=1`, revised, revised)
	if err != nil {
		t.Fatal(err)
	}
	reload := func(revision uint64, version int) {
		t.Helper()
		if _, err := service.studio.ExecContext(ctx, "UPDATE component_publications SET active_version_no=? WHERE report_id='records'", version); err != nil {
			t.Fatal(err)
		}
		if err := manager.Reload(ctx, application.Request{Revision: revision, Compile: func(ctx context.Context, seed *typecatalog.Catalog) (*application.Build, error) {
			return service.compile(ctx, seed, nil)
		}}); err != nil {
			t.Fatal(err)
		}
	}
	check := func(callContext context.Context, expected string) {
		t.Helper()
		result, err := manager.InvokeComponent(callContext, dexec.ComponentRequest{Target: parent})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(result.(*composingOutput).Data, expected) {
			t.Fatalf("expected %s, got %+v", expected, result)
		}
	}
	reload(2, 2)
	check(ctx, "revision-two")
	check(pinned, "alpha-101")
	reload(3, 1)
	check(ctx, "alpha-101")
	foreign := dexec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: "example.com/runtime/foreign", Name: "foreign"}, Route: spec.RouteRef{Method: http.MethodGet, Path: "/foreign"}}
	if _, err = manager.InvokeComponent(ctx, dexec.ComponentRequest{Target: foreign}); err == nil {
		t.Fatal("unregistered target resolved")
	}
}

func TestNamespaceBuildResolverUsesCurrentIdentityAndClosesFailedBuild(t *testing.T) {
	ctx := context.Background()
	active := "first"
	resolved := []string{}
	composed := []string{}
	closed := 0
	unavailable := errors.New("configuration unavailable")
	service := &Service{config: Config{NamespaceBuildResolver: func(_ context.Context, id string) (NamespaceBuildExtension, error) {
		resolved = append(resolved, id)
		if id == "failed" {
			return nil, unavailable
		}
		if id != active {
			return nil, nil
		}
		return func(context.Context, *application.Build) error { composed = append(composed, id); return nil }, nil
	}}}
	for _, id := range []string{"", "generic", "first", "second"} {
		service.config.NamespaceID = id
		build := &application.Build{}
		got, err := service.extendNamespaceBuild(ctx, build)
		if err != nil || got != build {
			t.Fatalf("%s: %v", id, err)
		}
	}
	active = "second"
	service.config.NamespaceID = "second"
	if _, err := service.extendNamespaceBuild(ctx, &application.Build{}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(resolved, ",") != "generic,first,second,second" || strings.Join(composed, ",") != "first,second" {
		t.Fatalf("resolved=%v composed=%v", resolved, composed)
	}
	service.config.NamespaceID = "failed"
	got, err := service.extendNamespaceBuild(ctx, &application.Build{Shutdown: func(context.Context) error { closed++; return nil }})
	if got != nil || !errors.Is(err, unavailable) || closed != 1 {
		t.Fatalf("failed lifecycle: %v %v %d", got, err, closed)
	}
	// Explicit startup authority wins, without calling mutable discovery twice.
	service.config.NamespaceBuildExtensions = map[string]NamespaceBuildExtension{"failed": func(context.Context, *application.Build) error { return nil }}
	before := len(resolved)
	if _, err := service.extendNamespaceBuild(ctx, &application.Build{}); err != nil || len(resolved) != before {
		t.Fatal("explicit extension did not retain precedence")
	}
}

func TestNamespaceNativeBuildResolverReceivesScopedCapability(t *testing.T) {
	ctx := context.Background()
	service := &Service{config: Config{NamespaceID: "owned-namespace"}}
	build := &application.Build{}
	called := false
	service.config.NamespaceBuildResolver = func(context.Context, string) (NamespaceBuildExtension, error) {
		t.Fatal("ordinary resolver displaced native assembly")
		return nil, nil
	}
	service.config.NamespaceNativeBuildResolver = func(callCtx context.Context, id string, native NativeComponentRuntime) (NamespaceBuildExtension, error) {
		if callCtx != ctx || id != "owned-namespace" || native != service {
			t.Fatal("native assembly received another namespace capability")
		}
		return func(context.Context, *application.Build) error { called = true; return nil }, nil
	}
	actual, err := service.extendNamespaceBuild(ctx, build)
	if err != nil || actual != build || !called {
		t.Fatalf("native namespace assembly: %v", err)
	}
}
