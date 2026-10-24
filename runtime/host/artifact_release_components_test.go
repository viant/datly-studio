package host

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/viant/forge/backend/artifactcohort"
	"github.com/viant/forge/backend/types"
	windowprotocol "github.com/viant/forge/backend/window/contract"
)

func releaseWindowFixture(t *testing.T, pin windowprotocol.ComponentBinding) *artifactcohort.Artifact {
	t.Helper()
	descriptor, err := types.CanonicalWindowDescriptor(json.RawMessage(`{"id":"records","backend":{"kind":"datly","ownership":"provider","method":"windows/datasource","schemaFingerprint":"` + pin.SchemaFingerprint + `","component":{"kind":"` + pin.Kind + `","id":"` + pin.ID + `","revision":"` + pin.Revision + `","schemaFingerprint":"` + pin.SchemaFingerprint + `","contentFingerprint":"` + pin.ContentFingerprint + `"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := types.WindowDescriptorFingerprint(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	variant := types.WindowResourceVariant{Window: &types.Window{WindowKey: "home", View: types.View{Content: &types.Container{ID: "table"}}, DataSource: map[string]types.DataSource{"records": {}}, ResourceDependencies: map[string]string{"records": fingerprint}}, DataSources: map[string]json.RawMessage{"records": descriptor}}
	variantHash, err := types.WindowVariantFingerprint(variant)
	if err != nil {
		t.Fatal(err)
	}
	envelope := types.WindowResourceEnvelope{SchemaVersion: 2, Format: types.WindowBundleFormat, Targets: []types.WindowTargetBinding{{Variant: variantHash}}, Variants: map[string]types.WindowResourceVariant{variantHash: variant}}
	if err = envelope.Validate(); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	return &artifactcohort.Artifact{Root: artifactcohort.RootRef{URI: "window://demo/home", Kind: "window", Version: 1, Hash: artifactcohort.Hash(data)}, Payload: data}
}

func TestReleaseWindowBindingUsesValidatedNativePin(t *testing.T) {
	pin := windowprotocol.ComponentBinding{Kind: "dynamic", ID: "records", Revision: "1", SchemaFingerprint: strings.Repeat("a", 64), ContentFingerprint: strings.Repeat("b", 64)}
	artifact := releaseWindowFixture(t, pin)
	configured := map[string]ComponentReference{"records": {Kind: "dynamic", ID: "records", Revision: "1", Method: http.MethodGet, Route: "/records"}}
	called := 0
	resolve := func(_ context.Context, ref ComponentReference) (windowprotocol.ComponentBinding, error) {
		called++
		if ref != configured["records"] {
			t.Fatalf("unexpected route mapping: %+v", ref)
		}
		return pin, nil
	}
	bindings, err := CompileReleaseWindowBindings(context.Background(), artifact, configured, resolve)
	if err != nil || len(bindings) != 1 || bindings[0].Root != artifact.Root || bindings[0].Reference != configured["records"] || called != 1 {
		t.Fatalf("binding %+v err=%v called=%d", bindings, err, called)
	}
	wrong := map[string]ComponentReference{"records": {Kind: "dynamic", ID: "other", Revision: "1", Method: http.MethodGet, Route: "/other"}}
	if _, err = CompileReleaseWindowBindings(context.Background(), artifact, wrong, resolve); err == nil {
		t.Fatal("unrelated component mapping accepted")
	}
	if _, err = CompileReleaseWindowBindings(context.Background(), artifact, configured, func(context.Context, ComponentReference) (windowprotocol.ComponentBinding, error) {
		bad := pin
		bad.SchemaFingerprint = strings.Repeat("c", 64)
		return bad, nil
	}); err == nil {
		t.Fatal("drifted native schema accepted")
	}
	bad := *artifact
	bad.Root.Hash = strings.Repeat("d", 64)
	if _, err = CompileReleaseWindowBindings(context.Background(), &bad, configured, resolve); err == nil {
		t.Fatal("corrupt released artifact accepted")
	}
}
