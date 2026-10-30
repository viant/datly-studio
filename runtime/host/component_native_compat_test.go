package host

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly/exec"
	mcptool "github.com/viant/datly/mcp/tool"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/transcribe"
	windowprotocol "github.com/viant/forge/backend/window/contract"
)

const legacyPrimaryFamilyDQL = `#package('example.com/runtime/records')
#setting($_ = $connector('source'))
#setting($_ = $route('/records','GET'))
#setting($_ = $case_format('lc'))
#setting($_ = $cube())
#setting($_ = $cubeCompose(true,false,3,100))
#define($_ = $Data<?>(output/view))
SELECT records.*,CAST(records.units AS int),tag(records.project_id,'json:"projectId" groupable:"true"'),tag(records.units,'json:"units" groupable:"false"'),selector_fields(records,true),selector_limit(records,true),selector_offset(records,true),selector_order_by(records,true),grouping_enabled(records)
FROM (SELECT t.project_id,SUM(t.id) AS units FROM tasks t GROUP BY t.project_id) records`

// A stable synthetic DSN makes the pre-change executable-source fingerprint
// portable across test directories. No live store or credential is used.
func legacyPrimaryFixture(t *testing.T) (*scopedHost, context.Context) {
	fixture, err := newScopedHost(t, map[string]string{"records": legacyPrimaryFamilyDQL}, map[string]authz.Policy{"records": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := oauth.WithBearer(context.Background(), fixture.token(t, "alice", []authz.Entity{}))
	source, err := sql.Open("sqlite", "file:legacy-primary-component-pin-v1?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { source.Close() })
	_, err = source.ExecContext(ctx, `CREATE TABLE tasks(id INTEGER PRIMARY KEY, project_id INTEGER NOT NULL, name TEXT NOT NULL);INSERT INTO tasks VALUES(1,101,'alpha'),(2,102,'beta')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = fixture.service.studio.ExecContext(ctx, `UPDATE connectors SET dsn_template='file:legacy-primary-component-pin-v1?mode=memory&cache=shared' WHERE name='source'`)
	if err != nil {
		t.Fatal(err)
	}
	store, err := authz.NewStaticStore([]authz.Document{{Resource: authz.Resource{Kind: "component", ID: "records", Version: "1", Tenant: scopeTestTenant}, Revision: 1, Policies: map[string]authz.Policy{"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}, "execute": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	fixture.service.resourceAccess.Store = store
	return fixture, ctx
}

// Filled from actual SDK 676e35c with its original Datly dependency. This is not
// calculated with the new family implementation inside the compatibility test.
const preFamilyPrimaryPinJSON = `{"kind":"dynamic","id":"records","revision":"1","contentFingerprint":"a3edaf4c6fb590191a3cf79e73637843f4e80160e8e7136378e485c9fb94920d","schemaFingerprint":"7e173c0f0219f49a770c93178626290a49e99910962590c2b66e70e5bade50d4"}`

func TestLegacyPrimaryJSONPinDoesNotGrantNativeFamily(t *testing.T) {
	fixture, ctx := legacyPrimaryFixture(t)
	ref := ComponentReference{Kind: "dynamic", ID: "records", Revision: "1", Method: http.MethodGet, Route: "/records"}
	var legacy windowprotocol.ComponentBinding
	if err := json.Unmarshal([]byte(preFamilyPrimaryPinJSON), &legacy); err != nil {
		t.Fatal(err)
	}
	observed, err := fixture.service.ResolveComponentBinding(ctx, ref)
	if err != nil || observed != legacy {
		t.Fatalf("legacy primary identity changed: %+v %v", observed, err)
	}
	raw, err := fixture.service.ExecuteComponentJSON(ctx, ref, legacy, nil)
	if err != nil || !strings.Contains(string(raw), "101") || !strings.Contains(string(raw), "102") {
		t.Fatalf("legacy primary JSON stopped working: %s %v", raw, err)
	}
	called := false
	execute := func(ctx context.Context, family NativeComponentFamily, invoker exec.ComponentInvoker) (any, error) {
		called = true
		for _, member := range family.Members() {
			for _, route := range member.Component.Routes {
				if route.Path != "/records/cube" {
					continue
				}
				target := exec.ComponentTarget{Component: member.Component.Key, Route: spec.RouteRef{Method: route.Method, Path: route.Path}}
				contract, _ := member.Input.ForRoute(target.Route)
				plan, err := mcptool.NewCompiler().Compile(mcptool.Input{Component: target.Component, Exposure: &spec.MCPExposure{Kind: spec.MCPExposureTool, Name: "compat.cube"}, Contract: contract, OutputType: member.OutputType, Output: member.Output, TransportReady: true})
				if err != nil {
					return nil, err
				}
				scope, err := plan.Scope(map[string]any{"dimensions": map[string]bool{"projectId": true}, "measures": map[string]bool{"units": true}, "filters": map[string]any{}, "limit": 10})
				if err != nil {
					return nil, err
				}
				defer scope.Close()
				return invoker.InvokeComponent(ctx, exec.ComponentRequest{Target: target, Providers: scope.Providers()})
			}
		}
		t.Fatal("cube missing")
		return nil, nil
	}
	if output, err := fixture.service.ExecuteNativeComponent(ctx, ref, NativeComponentBinding{ComponentBinding: legacy}, execute); err == nil || output != nil || called {
		t.Fatal("legacy primary pin gained sibling authority")
	}
	nativePin, err := fixture.service.ResolveNativeComponentBinding(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	if nativePin.ComponentBinding != legacy || nativePin.FamilyFingerprint == "" {
		t.Fatal("native pin replaced the legacy primary identity")
	}
	if output, err := fixture.service.ExecuteNativeComponent(ctx, ref, nativePin, execute); err != nil || output == nil {
		t.Fatalf("explicit native family pin failed: %v", err)
	}
	stale := nativePin
	stale.FamilyFingerprint = strings.Repeat("a", 64)
	called = false
	if output, err := fixture.service.ExecuteNativeComponent(ctx, ref, stale, execute); err == nil || output != nil || called {
		t.Fatal("mismatched family proof executed")
	}
	// Model a compiler/runtime upgrade that changes only generated composition
	// metadata. The source bytes and primary schema remain exactly unchanged.
	priorCache := fixture.service.compiledArtifacts
	fixture.service.compiledArtifacts = nil
	budget := 100
	fixture.service.compileRuntimeContract = func(ctx context.Context, source *transcribe.Source) (*transcribe.RuntimeContract, error) {
		contract, err := transcribe.NewCompiler().RuntimeContracts(ctx, fixture.service.config.RootDir, source)
		if err != nil {
			return nil, err
		}
		contract.Component.Settings.Report.Compose.MaxLimit = budget
		return contract, nil
	}
	t.Cleanup(func() { fixture.service.compiledArtifacts = priorCache; fixture.service.compileRuntimeContract = nil })
	// A sibling change during a native call is checked on the existing completion
	// rebuild, even though the ordinary primary-only completion pin still matches.
	output, err := fixture.service.ExecuteNativeComponent(ctx, ref, nativePin, func(ctx context.Context, family NativeComponentFamily, invoker exec.ComponentInvoker) (any, error) {
		result, err := execute(ctx, family, invoker)
		if err == nil {
			budget = 50
		}
		return result, err
	})
	if err == nil || output != nil || !strings.Contains(err.Error(), "family binding changed") {
		t.Fatalf("family completion drift leaked output: %v %v", output, err)
	}
	currentPrimary, err := fixture.service.ResolveComponentBinding(ctx, ref)
	if err != nil || currentPrimary != legacy {
		t.Fatalf("sibling-only change invalidated legacy primary pin: %+v %v", currentPrimary, err)
	}
	raw, err = fixture.service.ExecuteComponentJSON(ctx, ref, legacy, nil)
	if err != nil || !strings.Contains(string(raw), "101") {
		t.Fatalf("legacy JSON changed with sibling metadata: %s %v", raw, err)
	}
	currentNative, err := fixture.service.ResolveNativeComponentBinding(ctx, ref)
	if err != nil || currentNative.ComponentBinding != legacy || currentNative.FamilyFingerprint == nativePin.FamilyFingerprint {
		t.Fatalf("native-only pin failed to isolate drift: %+v %v", currentNative, err)
	}
	called = false
	if output, err := fixture.service.ExecuteNativeComponent(ctx, ref, nativePin, execute); err == nil || output != nil || called {
		t.Fatal("stale family pin was accepted after sibling-only change")
	}
	if output, err := fixture.service.ExecuteNativeComponent(ctx, ref, currentNative, execute); err != nil || output == nil {
		t.Fatalf("fresh native family pin failed: %v", err)
	}

}
