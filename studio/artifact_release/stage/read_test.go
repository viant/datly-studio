package stage_test

import (
	"context"
	bundle "github.com/viant/datly-studio/studio/artifact_release/bundle"
	snapshot "github.com/viant/datly-studio/studio/artifact_release/snapshot"
	"github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
	"reflect"
	"testing"
)

func TestNativeSnapshotAndImmutableBundleReads(t *testing.T) {
	ctx := context.Background()
	_, server, stage := newStageCase(ctx, t, "")
	if err := invokeStage(ctx, server, stage); err != nil {
		t.Fatal(err)
	}
	input := &snapshot.Input{}
	input.SetScopeId(stage.ScopeId)
	input.SetName(stage.Name)
	read := func() *snapshot.Output {
		t.Helper()
		value, err := server.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[snapshot.SnapshotComponent]().PkgPath(), Name: "snapshot"}, Route: spec.RouteRef{Method: "GET", Path: "/_studio/artifact-release-store/snapshot"}}, Input: input})
		if err != nil {
			t.Fatal(err)
		}
		return value.(*snapshot.Output)
	}
	output := read()
	if len(output.Data) != 1 || output.Data[0].Operation == nil || *output.Data[0].Operation.OperationId != *stage.Stage[0].PendingOperationId || output.Data[0].Revision != 0 || output.Data[0].ActiveDependenciesJson != nil {
		t.Fatalf("wrong native pending snapshot: %+v", output.Data)
	}
	input.SetScopeId("foreign")
	if output = read(); len(output.Data) != 0 {
		t.Fatal("foreign scope disclosed channel")
	}
	bi := &bundle.Input{}
	bi.SetScopeId(stage.ScopeId)
	bi.SetBundleHash(stage.TargetBundleHash)
	value, err := server.InvokeComponent(ctx, exec.ComponentRequest{Target: exec.ComponentTarget{Component: spec.Key{Kind: spec.KindComponent, Scope: reflect.TypeFor[bundle.BundleComponent]().PkgPath(), Name: "bundle"}, Route: spec.RouteRef{Method: "GET", Path: "/_studio/artifact-release-store/bundle"}}, Input: bi})
	if err != nil {
		t.Fatal(err)
	}
	bo := value.(*bundle.Output)
	if len(bo.Data) != 1 || bo.Data[0].Binding == nil || *bo.Data[0].Binding.BindingsHash != stage.BindingsHash {
		t.Fatal("immutable binding snapshot missing")
	}
}
