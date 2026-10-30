package host

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
)

// This fixture exercises real RSA verification, version-scoped access-context
// binding and SQL entity predicates. It deliberately has no business adapter.
func TestNativeExactComponentInternalValidatedVersion(t *testing.T) {
	dql := strings.Replace(scopedTasksDQL, "#setting($_ = $mcp('tasks.list','List tasks'))", "#setting($_ = $internal(true))", 1)
	policy := authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}, EntityType: "project"}
	fixture, err := newScopedHost(t, map[string]string{"tasks": dql}, map[string]authz.Policy{"tasks": policy})
	if err != nil {
		t.Fatal(err)
	}
	service := fixture.service
	entities := []authz.Entity{{Type: "project", ID: "101"}}
	token := fixture.token(t, "alice", entities)
	ctx := oauth.WithBearer(context.Background(), token)
	documents := []authz.Document{}
	for _, version := range []string{"1", "2"} {
		documents = append(documents, authz.Document{Resource: authz.Resource{Kind: "component", ID: "tasks", Version: version, Tenant: scopeTestTenant}, Revision: 1, Policies: map[string]authz.Policy{"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}, "execute": policy}})
	}
	store, err := authz.NewStaticStore(documents)
	if err != nil {
		t.Fatal(err)
	}
	service.resourceAccess.Store = store
	draft := strings.Replace(dql, "t.name", "'draft-v2' AS name", 1)
	_, err = service.studio.ExecContext(ctx, `INSERT INTO component_versions(report_id,version_no,state,authoring_mode,authored_dql,generated_dql,component_spec_json,spec_format_version,spec_hash,type_manifest_json,compile_status,datly_version,compiler_version,source_revision,created_by,created_at)
 SELECT report_id,2,'draft',authoring_mode,?,?,component_spec_json,spec_format_version,'hash-v2',type_manifest_json,compile_status,datly_version,compiler_version,2,created_by,created_at FROM component_versions WHERE report_id='tasks' AND version_no=1`, draft, draft)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.studio.ExecContext(ctx, `UPDATE component_versions SET validated_at=CURRENT_TIMESTAMP WHERE report_id='tasks' AND version_no=2`)
	if err != nil {
		t.Fatal(err)
	}
	ref := ComponentReference{Kind: "dynamic", ID: "tasks", Revision: "2", Method: http.MethodGet, Route: "/tasks"}
	pin, err := service.ResolveNativeComponentBinding(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}

	if err := service.PrepareNativeComponent(context.Background(), ref, pin, func(family NativeComponentFamily) error {
		metadata := family.Primary()
		if metadata.Reader != nil || metadata.Handler != nil || metadata.DataSource != nil || len(metadata.Providers) != 0 || !reflect.DeepEqual(metadata.Capabilities, registry.RegisteredComponent{}.Capabilities) {
			t.Fatal("startup inspection retained executable capability")
		}
		if metadata.Input == nil || metadata.Output == nil {
			t.Fatal("startup metadata missing contracts")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	cancelledInspection, cancelInspection := context.WithCancel(ctx)
	if err := service.PrepareNativeComponent(cancelledInspection, ref, pin, func(NativeComponentFamily) error { cancelInspection(); return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled inspection reported success: %v", err)
	}
	// Merely setting compile_status without successful persisted validation is
	// insufficient for an SDK-lifecycle draft, including after validation reset.
	_, err = service.studio.ExecContext(ctx, `UPDATE component_versions SET validated_at=NULL WHERE report_id='tasks' AND version_no=2`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ResolveNativeComponentBinding(ctx, ref); err == nil {
		t.Fatal("unvalidated draft resolved")
	}
	_, err = service.studio.ExecContext(ctx, `UPDATE component_versions SET validated_at=CURRENT_TIMESTAMP WHERE report_id='tasks' AND version_no=2`)
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{"/tasks", "/_studio/access/context/tasks/project"} {
		status, body := fixture.get(t, route, token, nil)
		if status < 400 || strings.Contains(body, "alpha-101") {
			t.Fatalf("internal route exposed: %s %d %s", route, status, body)
		}
	}
	_, toolBody := fixture.callTool(t, token, "tasks.list", nil)
	if strings.Contains(toolBody, "alpha-101") || (!strings.Contains(toolBody, "error") && !strings.Contains(toolBody, "isError")) {
		t.Fatalf("internal tool exposed: %s", toolBody)
	}
	if out, err := service.ExecuteComponentJSON(ctx, ref, pin.ComponentBinding, nil); err == nil || len(out) != 0 {
		t.Fatalf("internal reader exposed over HTTP: %s %v", out, err)
	}
	var retained exec.ComponentInvoker
	var request exec.ComponentRequest
	invoke := func(ctx context.Context, family NativeComponentFamily, invoker exec.ComponentInvoker) (any, error) {
		registered := family.Primary()
		if registered.Reader != nil || registered.Handler != nil || registered.DataSource != nil || len(registered.Providers) != 0 || !reflect.DeepEqual(registered.Capabilities, registry.RegisteredComponent{}.Capabilities) {
			t.Fatal("execution metadata leaked direct execution capability")
		}
		retained = invoker
		request = exec.ComponentRequest{Target: exec.ComponentTarget{Component: registered.Component.Key, Route: spec.RouteRef{Method: ref.Method, Path: ref.Route}}}
		wrong := request
		wrong.Target.Route.Path = "/_studio/access/context/tasks/project"
		if output, err := invoker.InvokeComponent(ctx, wrong); err == nil || output != nil {
			t.Fatal("native capability accepted another target")
		}
		return invoker.InvokeComponent(ctx, request)
	}
	output, err := service.ExecuteNativeComponent(ctx, ref, pin, invoke)
	raw, _ := json.Marshal(output)
	if err != nil || !strings.Contains(string(raw), "draft-v2") || strings.Contains(string(raw), "102") || strings.Contains(string(raw), "103") || strings.Contains(string(raw), "104") {
		t.Fatalf("native scoped draft output=%s err=%v", raw, err)
	}
	escaped, ok := retained.(*exactNativeInvoker)
	if !ok || escaped.manager != nil || escaped.metadata != nil || escaped.authority != nil {
		t.Fatal("closed invoker retained generation capabilities")
	}
	if output, err := retained.InvokeComponent(ctx, request); err == nil || output != nil {
		t.Fatal("native capability escaped completion")
	}
	for label, bearer := range map[string]string{"invalid": "invalid", "expired": fixture.expiredToken(t, "alice", entities), "unassigned": fixture.token(t, "alice", nil)} {
		denied := oauth.WithBearer(context.Background(), bearer)
		if output, err := service.ExecuteNativeComponent(denied, ref, pin, invoke); err == nil || output != nil {
			t.Fatalf("%s credential released data: %v %v", label, output, err)
		}
	}
	baseline := ref
	baseline.Revision = "1"
	basePin, err := service.ResolveNativeComponentBinding(ctx, baseline)
	if err != nil {
		t.Fatal(err)
	}
	output, err = service.ExecuteNativeComponent(ctx, baseline, basePin, invoke)
	raw, _ = json.Marshal(output)
	if err != nil || !strings.Contains(string(raw), "alpha-101") || strings.Contains(string(raw), "draft-v2") {
		t.Fatalf("baseline changed: %s %v", raw, err)
	}
	var active, generation int
	if err = service.studio.QueryRowContext(ctx, `SELECT active_version_no,active_generation FROM component_publications WHERE report_id='tasks'`).Scan(&active, &generation); err != nil || active != 1 || generation != 1 {
		t.Fatalf("publication changed %d %d %v", active, generation, err)
	}
	var state string
	if err = service.studio.QueryRowContext(ctx, `SELECT state FROM component_versions WHERE report_id='tasks' AND version_no=2`).Scan(&state); err != nil || state != "draft" {
		t.Fatalf("draft published: %s %v", state, err)
	}
	// Source changes during native execution must suppress its completed output.
	output, err = service.ExecuteNativeComponent(ctx, ref, pin, func(ctx context.Context, family NativeComponentFamily, i exec.ComponentInvoker) (any, error) {
		result, e := invoke(ctx, family, i)
		if e != nil {
			return nil, e
		}
		_, e = service.studio.ExecContext(ctx, `UPDATE component_versions SET generated_dql=? WHERE report_id='tasks' AND version_no=2`, strings.Replace(draft, "draft-v2", "changed-v2", 1))
		return result, e
	})
	if err == nil || output != nil {
		t.Fatalf("completion source drift released data: %v %v", output, err)
	}
}

func TestNativeExactRejectsUnboundEntityPolicy(t *testing.T) {
	dql := strings.Replace(unscopedRecordsDQL, "#define($_ = $Records", "#setting($_ = $internal(true))\n#define($_ = $Records", 1)
	policy := authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}, EntityType: "project"}
	fixture, err := newScopedHost(t, map[string]string{"records": dql}, map[string]authz.Policy{"records": policy})
	if err != nil {
		t.Fatal(err)
	}
	ctx := oauth.WithBearer(context.Background(), fixture.token(t, "alice", []authz.Entity{{Type: "project", ID: "101"}}))
	store, err := authz.NewStaticStore([]authz.Document{{Resource: authz.Resource{Kind: "component", ID: "records", Version: "1", Tenant: scopeTestTenant}, Revision: 1, Policies: map[string]authz.Policy{"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}, "execute": policy}}})
	if err != nil {
		t.Fatal(err)
	}
	fixture.service.resourceAccess.Store = store
	ref := ComponentReference{Kind: "dynamic", ID: "records", Revision: "1", Method: http.MethodGet, Route: "/records"}
	pin, err := fixture.service.ResolveNativeComponentBinding(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	output, err := fixture.service.ExecuteNativeComponent(ctx, ref, pin, func(ctx context.Context, family NativeComponentFamily, i exec.ComponentInvoker) (any, error) {
		return i.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{Component: family.Primary().Component.Key, Route: spec.RouteRef{Method: ref.Method, Path: ref.Route}}})
	})
	if err == nil || output != nil {
		t.Fatalf("unbound entity policy released data: %v %v", output, err)
	}
}

func TestPrepareNativeComponentCleanupPreservesValuesAndBoundsCancellation(t *testing.T) {
	type cleanupValueKey struct{}
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), cleanupValueKey{}, "owned-cleanup"))
	cancel()
	cleanupFailure := errors.New("manager shutdown failed")
	observed := false
	err := shutdownNativeMetadata(ctx, func(cleanup context.Context) error {
		observed = true
		deadline, bounded := cleanup.Deadline()
		if cleanup.Value(cleanupValueKey{}) != "owned-cleanup" || cleanup.Err() != nil || !bounded || time.Until(deadline) <= 0 || time.Until(deadline) > 5*time.Second {
			t.Errorf("cleanup lost caller values or deadline: bounded=%v err=%v", bounded, cleanup.Err())
		}
		return cleanupFailure
	})
	if !observed || !errors.Is(err, cleanupFailure) {
		t.Fatalf("cleanup result=%v observed=%v", err, observed)
	}
}

func TestNativeComponentDescriptionStripsExecutionAndClonesSpec(t *testing.T) {
	source := exactDependency(t)
	if source.Handler == nil {
		t.Fatal("fixture must carry an actual native handler")
	}
	first, err := nativeComponentDescription(source)
	if err != nil {
		t.Fatal(err)
	}
	second, err := nativeComponentDescription(source)
	if err != nil {
		t.Fatal(err)
	}
	if first.Reader != nil || first.Handler != nil || first.DataSource != nil || len(first.Providers) != 0 || !reflect.DeepEqual(first.Capabilities, registry.RegisteredComponent{}.Capabilities) {
		t.Fatal("description copied execution carriers")
	}
	originalKey, originalPath := source.Component.Key, source.Component.Routes[0].Path
	first.Component.Key.Name = "inspection-only"
	first.Component.Routes[0].Path = "/inspection-only"
	if source.Component.Key != originalKey || source.Component.Routes[0].Path != originalPath || second.Component.Key != originalKey || second.Component.Routes[0].Path != originalPath {
		t.Fatal("description mutated source metadata")
	}
	if first.Input != source.Input || first.Output != source.Output {
		t.Fatal("canonical immutable binding/encoding plans were recompiled")
	}
}

// Interfaces hidden in linked-source metadata must become ordinary wire values,
// never retain an invocation object in a callback's copied Component spec.
func TestNativeComponentDescriptionDropsNestedInvocationCarrier(t *testing.T) {
	source := exactDependency(t)
	source.Component.Parameters = append(source.Component.Parameters, &spec.Parameter{Name: "metadata", Predicates: []*spec.Predicate{{HandlerArgs: []any{&exactNativeInvoker{}}}}})
	described, err := nativeComponentDescription(source)
	if err == nil || described != nil {
		t.Fatal("nested invocation carrier was accepted as scalar metadata")
	}
}
