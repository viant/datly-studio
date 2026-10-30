package host

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly/exec"
	mcptool "github.com/viant/datly/mcp/tool"
	native "github.com/viant/datly/report"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	identity "github.com/viant/forge/backend/resource"
)

const nativeFamilyDQL = `#package('example.com/runtime/scoped')
#import('studioaccess','github.com/viant/datly-studio/runtime/accesscontext')
#setting($_ = $connector('source'))
#setting($_ = $route('/tasks','GET'))
#setting($_ = $case_format('lc'))
#setting($_ = $internal(true))
#setting($_ = $cube())
#setting($_ = $cubeCompose(true,false,3,100))
#define($_ = $Auth<*studioaccess.Output>(component/GET:/_studio/access/context/tasks/project).Required())
#define($_ = $ProjectIDs<[]string,[]int>(param/Auth.Scope.IDs).WithCodec('EntityIDs').Required().WithPredicate(0,'in','t','project_id'))
#define($_ = $Data<?>(output/view))
SELECT tasks.*,CAST(tasks.units AS int),tag(tasks.project_id,'json:"projectId" groupable:"true"'),tag(tasks.units,'json:"units" groupable:"false"'),selector_fields(tasks,true),selector_limit(tasks,true),selector_offset(tasks,true),selector_order_by(tasks,true),grouping_enabled(tasks)
FROM (SELECT t.project_id,SUM(t.id)*1 AS units FROM tasks t ${predicate.Builder().CombineAnd($predicate.FilterGroup(0,"AND")).Build("WHERE")} GROUP BY t.project_id) tasks`

func TestNativeExactFamilyCubeComposeAndSourceConfinement(t *testing.T) {
	policy := authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}, EntityType: "project"}
	fixture, err := newScopedHost(t, map[string]string{"tasks": nativeFamilyDQL, "records": unscopedRecordsDQL}, map[string]authz.Policy{"tasks": policy, "records": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}})
	if err != nil {
		t.Fatal(err)
	}
	service := fixture.service
	ctx := oauth.WithBearer(context.Background(), fixture.token(t, "alice", []authz.Entity{{Type: "project", ID: "101"}}))
	var documents []authz.Document
	for _, version := range []string{"1", "2"} {
		documents = append(documents, authz.Document{Resource: authz.Resource{Kind: "component", ID: "tasks", Version: version, Tenant: scopeTestTenant}, Revision: 1, Policies: map[string]authz.Policy{"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}, "execute": policy}})
	}
	documents = append(documents, authz.Document{Resource: authz.Resource{Kind: "component", ID: "records", Version: "1", Tenant: scopeTestTenant}, Revision: 1, Policies: map[string]authz.Policy{"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}, "execute": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}}})
	store, err := authz.NewStaticStore(documents)
	if err != nil {
		t.Fatal(err)
	}
	service.resourceAccess.Store = store
	draft := strings.Replace(nativeFamilyDQL, "SUM(t.id)*1", "SUM(t.id)*10", 1)
	_, err = service.studio.ExecContext(ctx, `INSERT INTO component_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at,validated_at)
 SELECT report_id,2,'draft',authoring_mode,?,?,component_spec_json,spec_format_version,'hash-v2',type_manifest_json,compile_status,datly_version,compiler_version,2,created_by,created_at,CURRENT_TIMESTAMP FROM component_versions WHERE report_id='tasks' AND version_no=1`, draft, draft)
	if err != nil {
		t.Fatal(err)
	}
	var supportTarget exec.ComponentTarget
	service.config.ExactBuildDependencyResolver = func(context.Context, string, ComponentReference) (*ExactBuildDependencies, error) {
		support := exactDependency(t)
		support.Component.Key.Scope = "example.com/runtime/scoped"
		supportTarget = exec.ComponentTarget{Component: support.Component.Key, Route: spec.RouteRef{Method: "GET", Path: "/support-context"}}
		return &ExactBuildDependencies{Components: []*registry.RegisteredComponent{support}, ContentFingerprint: identity.ContentFingerprint([]byte("support-v1"))}, nil
	}
	ref := ComponentReference{Kind: "dynamic", ID: "tasks", Revision: "2", Method: http.MethodGet, Route: "/tasks"}
	pin, err := service.ResolveNativeComponentBinding(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	otherRef := ComponentReference{Kind: "dynamic", ID: "records", Revision: "1", Method: "GET", Route: "/records"}
	otherPin, err := service.ResolveComponentBinding(ctx, otherRef)
	if err != nil {
		t.Fatal(err)
	}
	independent, err := service.ExecuteComponentJSON(ctx, otherRef, otherPin, nil)
	if err != nil || !strings.Contains(string(independent), "alpha-101") || !strings.Contains(string(independent), "beta-102") {
		t.Fatalf("other source must be independently authorized: %s %v", independent, err)
	}
	definition := native.CompositionDefinition{SQL: "SELECT t1.project_id AS project,t1.units AS units FROM $CubeSQL1 AS t1 ORDER BY units DESC LIMIT 10", FrameCount: 1}
	var retained NativeComponentFamily
	var composeTarget exec.ComponentTarget
	if err = service.PrepareNativeComponent(context.Background(), ref, pin, func(family NativeComponentFamily) error {
		retained = family
		if len(family.Members()) != 3 || len(family.Compositions()) != 1 {
			t.Fatalf("wrong family inventory: %d %d", len(family.Members()), len(family.Compositions()))
		}
		for _, member := range family.Members() {
			if member.Reader != nil || member.Handler != nil || member.DataSource != nil || len(member.Providers) != 0 || !reflect.DeepEqual(member.Capabilities, registry.RegisteredComponent{}.Capabilities) {
				t.Fatal("family leaked execution capability")
			}
		}
		composition := family.Compositions()[0]
		composeTarget = composition.Target
		if composition.Source.Component != family.Primary().Component.Key || composition.Source.Route.Path != "/tasks" {
			t.Fatal("native compose source changed")
		}
		prepared, err := family.PrepareComposition(context.Background(), composeTarget, definition)
		if err != nil {
			return err
		}
		if prepared.SourceTarget() != composition.Source {
			t.Fatal("native planning source changed")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := retained.PrepareComposition(ctx, composeTarget, definition); err == nil {
		t.Fatal("family planner escaped callback lifetime")
	}
	if retained.Members() != nil {
		t.Fatal("closed family retained member registrations")
	}

	invoke := func(path string, arguments map[string]any) func(context.Context, NativeComponentFamily, exec.ComponentInvoker) (any, error) {
		return func(ctx context.Context, family NativeComponentFamily, invoker exec.ComponentInvoker) (any, error) {
			// Both a genuine other-source component and an injected support dependency
			// remain outside the allowlist, even though all live in this application.
			foreign := exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: "example.com/runtime/records", Name: "records"}, Route: spec.RouteRef{Method: "GET", Path: "/records"}}
			for _, denied := range []exec.ComponentTarget{foreign, supportTarget, {Component: spec.Key{Kind: spec.KindComponent, Scope: "example.com/runtime/scoped", Name: "unknown"}, Route: spec.RouteRef{Method: "POST", Path: "/tasks/cube/unknown"}}} {
				if result, err := invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: denied}); err == nil || result != nil || !strings.Contains(err.Error(), "native exact component target unavailable") {
					t.Fatalf("foreign target escaped family: %+v %v", denied, err)
				}
			}
			for _, member := range family.Members() {
				for _, route := range member.Component.Routes {
					if route.Path != path {
						continue
					}
					target := exec.ComponentTarget{Component: member.Component.Key, Route: spec.RouteRef{Method: route.Method, Path: route.Path}}
					contract, ok := member.Input.ForRoute(target.Route)
					if !ok {
						t.Fatal("family route contract missing")
					}
					plan, err := mcptool.NewCompiler().Compile(mcptool.Input{Component: target.Component, Exposure: &spec.MCPExposure{Kind: spec.MCPExposureTool, Name: "native.family"}, Contract: contract, OutputType: member.OutputType, Output: member.Output, TransportReady: true})
					if err != nil {
						return nil, err
					}
					scope, err := plan.Scope(arguments)
					if err != nil {
						return nil, err
					}
					defer scope.Close()
					return invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target, Providers: scope.Providers()})
				}
			}
			t.Fatal("family target unavailable")
			return nil, nil
		}
	}
	output, err := service.ExecuteNativeComponent(ctx, ref, pin, invoke("/tasks/cube", map[string]any{"dimensions": map[string]bool{"projectId": true}, "measures": map[string]bool{"units": true}, "filters": map[string]any{}, "limit": 10}))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(reflect.Indirect(reflect.ValueOf(output)).FieldByName("Data").Interface())
	if err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	if err = json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["projectId"] != float64(101) || rows[0]["units"] != float64(10) {
		t.Fatalf("scoped native cube=%s", raw)
	}
	output, err = service.ExecuteNativeComponent(ctx, ref, pin, invoke("/tasks/cube/compose", map[string]any{"sql": definition.SQL, "cubes": []any{map[string]any{"filters": map[string]any{}}}}))
	if err != nil {
		t.Fatal(err)
	}
	composed, ok := output.(*native.ComposeResponse)
	if !ok || !composed.Complete || composed.HasMore {
		t.Fatalf("native compose output=%#v", output)
	}
	values := reflect.ValueOf(composed.Data)
	if values.Len() != 1 {
		t.Fatalf("native compose rows=%d", values.Len())
	}
	row := reflect.Indirect(values.Index(0))
	if reflect.Indirect(row.FieldByName("Project")).Int() != 101 || reflect.Indirect(row.FieldByName("Units")).Int() != 10 {
		t.Fatalf("native compose scope=%#v", composed.Data)
	}
	// Stable inventory ordering and cold exact reconstruction must preserve pins.
	again, err := service.ResolveNativeComponentBinding(ctx, ref)
	if err != nil || again != pin {
		t.Fatalf("cold family pin changed: %v", err)
	}
	metadata := &exactComponentMetadata{}
	manager, _, err := service.componentRuntimeInspected(ctx, ref, true, metadata)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Shutdown(ctx)

	independentMetadata := &exactComponentMetadata{}
	independentManager, _, err := service.componentRuntimeInspected(ctx, ref, true, independentMetadata)
	if err != nil {
		t.Fatal(err)
	}
	defer independentManager.Shutdown(ctx)
	mixed := *metadata
	mixed.family = append([]*registry.RegisteredComponent(nil), metadata.family...)
	for i, member := range mixed.family {
		if _, ok := native.CompositionSource(member.Handler); !ok {
			continue
		}
		for _, foreign := range independentMetadata.family {
			if foreign.Component.Key == member.Component.Key {
				mixed.family[i] = foreign
				break
			}
		}
	}
	if _, err := newNativeComponentFamily(&mixed, ref); err == nil {
		t.Fatal("another compilation's compose handler was accepted by matching names")
	}
	if _, err := exactFamilyFingerprint(mixed.family); err == nil {
		t.Fatal("family fingerprint accepted foreign input-contract provenance")
	}
	before, err := exactFamilyFingerprint(metadata.family)
	if err != nil {
		t.Fatal(err)
	}
	reversed := append([]*registry.RegisteredComponent(nil), metadata.family...)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	ordered, err := exactFamilyFingerprint(reversed)
	if err != nil || before != ordered {
		t.Fatal("family order changed fingerprint")
	}
	for _, member := range metadata.family {
		if member != metadata.registered {
			member.Component.Description += " changed sibling contract"
			break
		}
	}
	changed, err := exactFamilyFingerprint(metadata.family)
	if err != nil || changed == before {
		t.Fatal("sibling metadata missing from family fingerprint")
	}
	// A source edit dropping generated siblings cannot reuse an earlier family pin.
	_, err = service.studio.ExecContext(ctx, `UPDATE component_versions SET generated_dql=? WHERE report_id='tasks' AND version_no=2`, strings.Replace(draft, "#setting($_ = $cubeCompose(true,false,3,100))", "", 1))
	if err != nil {
		t.Fatal(err)
	}
	if result, err := service.ExecuteNativeComponent(ctx, ref, pin, invoke("/tasks/cube", nil)); err == nil || result != nil {
		t.Fatal("changed generated family reused old pin")
	}
}

func TestNativeExactFamilyPreservesPerTargetBoundedAuthorization(t *testing.T) {
	var lines []string
	for _, line := range strings.Split(nativeFamilyDQL, "\n") {
		if !strings.Contains(line, "#define($_ = $Auth") && !strings.Contains(line, "#define($_ = $ProjectIDs") {
			lines = append(lines, line)
		}
	}
	policy := authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}, EntityType: "project"}
	fixture, err := newScopedHost(t, map[string]string{"tasks": strings.Join(lines, "\n")}, map[string]authz.Policy{"tasks": policy})
	if err != nil {
		t.Fatal(err)
	}
	ctx := oauth.WithBearer(context.Background(), fixture.token(t, "alice", []authz.Entity{{Type: "project", ID: "101"}}))
	store, err := authz.NewStaticStore([]authz.Document{{Resource: authz.Resource{Kind: "component", ID: "tasks", Version: "1", Tenant: scopeTestTenant}, Revision: 1, Policies: map[string]authz.Policy{"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}, "execute": policy}}})
	if err != nil {
		t.Fatal(err)
	}
	fixture.service.resourceAccess.Store = store
	ref := ComponentReference{Kind: "dynamic", ID: "tasks", Revision: "1", Method: "GET", Route: "/tasks"}
	pin, err := fixture.service.ResolveNativeComponentBinding(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	result, err := fixture.service.ExecuteNativeComponent(ctx, ref, pin, func(ctx context.Context, family NativeComponentFamily, invoker exec.ComponentInvoker) (any, error) {
		target := family.Compositions()[0].Target
		return invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target})
	})
	if err == nil || result != nil || !strings.Contains(err.Error(), "entity-bounded policy requires") {
		t.Fatalf("derived target lost compiled authorization: %v %v", result, err)
	}
}
