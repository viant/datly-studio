package artifactrelease

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/sdk/artifactrelease/releasegate"
	releasevalue "github.com/viant/datly-studio/sdk/artifactrelease/value"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
)

func TestComponentRootsPreparedIndexIsolationAndGenerationFence(t *testing.T) {
	ctx := t.Context()
	publisher := artifactcohort.NewPublisher()
	_, err := publisher.ReplaceCatalog(ctx, nil, nil, 0)
	require.NoError(t, err)
	scope := releasevalue.Scope{Tenant: "tenant", Account: "account", Workspace: "workspace", Namespace: "example"}
	root := func(name string) artifactcohort.RootRef {
		return artifactcohort.RootRef{Kind: "window", URI: "window://example/" + name, Version: 1, Hash: strings.Repeat("a", 64)}
	}
	pin := releasevalue.ComponentBinding{ComponentKind: "dynamic", ComponentID: "reader", ComponentRevision: "1", Method: "GET", Route: "/reader", SchemaFingerprint: strings.Repeat("b", 64), ContentFingerprint: strings.Repeat("c", 64)}
	selection := func(name string, ref artifactcohort.RootRef, source releasevalue.ComponentBinding) ActiveSelection {
		source.Root = ref
		source.DataSourceID = "rows"
		return ActiveSelection{Channel: contract.Channel{Key: contract.Key{Tenant: scope.Tenant, Account: scope.Account, Workspace: scope.Workspace, Namespace: scope.Namespace, Name: name}, Policies: []authz.NamedReference{{Resource: authz.Resource{Kind: "policy", ID: "audience", Version: "logical", Tenant: scope.Tenant}, Revision: 1}}}, Bundle: contract.Bundle{Namespace: scope.Namespace, Roots: []artifactcohort.RootRef{ref}}, Bindings: []releasevalue.ComponentBinding{source}}
	}
	original := []ActiveSelection{selection("denied", root("first"), pin), selection("allowed", root("second"), pin), selection("duplicate", root("second"), pin)}
	selected := original
	audienceCalls := 0
	c := &Controller{config: ControllerConfig{Scope: scope, Publisher: publisher, VerifyScope: func(context.Context) (releasevalue.Scope, error) { return scope, nil }, Authorize: func(context.Context, string) error { return nil }, AdmitAudience: func(_ context.Context, ch contract.Channel, _ authz.NamedReference) (bool, error) {
		audienceCalls++
		return ch.Key.Name == "allowed", nil
	}}}
	c.gate, err = releasegate.New(&activeGraphStore{}, releasegate.Projection{Reconcile: func(context.Context) error {
		manifest, err := publisher.Snapshot()
		if err == nil {
			c.setActive(manifest, selected)
		}
		return err
	}, Snapshot: func(context.Context) (any, error) { return nil, nil }, Project: func(context.Context, contract.PendingRevision) error { return nil }, Restore: func(context.Context, any) error { return nil }})
	require.NoError(t, err)
	require.NoError(t, c.gate.Recover(ctx))
	candidates, hash, err := c.ComponentRoots(ctx, pin)
	require.NoError(t, err)
	require.Equal(t, []artifactcohort.RootRef{root("first"), root("second")}, candidates)
	require.Zero(t, audienceCalls, "lookup must not grant access")
	_, err = c.AdmitRoot(ctx, candidates[0])
	require.Error(t, err)
	admitted, err := c.AdmitRoot(ctx, candidates[1])
	require.NoError(t, err)
	require.Equal(t, hash, admitted)
	candidates[0].URI = "window://example/forged"
	again, _, err := c.ComponentRoots(ctx, pin)
	require.NoError(t, err)
	require.Equal(t, root("first"), again[0])
	callerPin := pin
	callerPin.Root = root("forged")
	callerPin.DataSourceID = "forged"
	again, _, err = c.ComponentRoots(ctx, callerPin)
	require.NoError(t, err)
	require.Equal(t, []artifactcohort.RootRef{root("first"), root("second")}, again)
	for field := 0; field < 7; field++ {
		wrong := pin
		values := []*string{&wrong.ComponentKind, &wrong.ComponentID, &wrong.ComponentRevision, &wrong.Method, &wrong.Route, &wrong.SchemaFingerprint, &wrong.ContentFingerprint}
		*values[field] += "-other"
		found, _, err := c.ComponentRoots(ctx, wrong)
		require.NoError(t, err)
		require.Empty(t, found)
	}
	// Multiple active source revisions stay independently addressable; the
	// index is built from all release bindings, never only a current catalog.
	versionTwo := pin
	versionTwo.ComponentRevision = "2"
	selected = append(append([]ActiveSelection(nil), original...), selection("allowed", root("version-two"), versionTwo))
	require.NoError(t, c.gate.Recover(ctx))
	firstVersion, _, err := c.ComponentRoots(ctx, pin)
	require.NoError(t, err)
	require.Len(t, firstVersion, 2)
	secondVersion, _, err := c.ComponentRoots(ctx, versionTwo)
	require.NoError(t, err)
	require.Equal(t, []artifactcohort.RootRef{root("version-two")}, secondVersion)
	// Drifted source pins replace the index at the same publication boundary.
	changed := pin
	changed.ContentFingerprint = strings.Repeat("d", 64)
	selected = []ActiveSelection{selection("allowed", root("second"), changed)}
	require.NoError(t, c.gate.Recover(ctx))
	stale, _, err := c.ComponentRoots(ctx, pin)
	require.NoError(t, err)
	require.Empty(t, stale)
	current, _, err := c.ComponentRoots(ctx, changed)
	require.NoError(t, err)
	require.Equal(t, []artifactcohort.RootRef{root("second")}, current)
	// Rollback reconstructs the previous exact mapping, then withdrawal empties it.
	selected = original
	require.NoError(t, c.gate.Recover(ctx))
	restored, oldHash, err := c.ComponentRoots(ctx, pin)
	require.NoError(t, err)
	require.Len(t, restored, 2)
	newArtifact, generation, err := publisher.Publish(ctx, artifactcohort.RootRef{Kind: "prompt", URI: "prompt://example/new", Version: 1}, json.RawMessage(`{"text":"new"}`), func(context.Context, json.RawMessage) error { return nil })
	require.NoError(t, err)
	policy := authz.Resource{Tenant: scope.Tenant, Kind: "cohort", ID: "changed", Version: "logical"}
	_, err = publisher.ReplaceCatalog(ctx, []authz.Document{{Resource: policy, Revision: 1, Policies: map[string]authz.Policy{artifactcohort.PolicyAction: {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}}}}, []artifactcohort.Cohort{{Tenant: scope.Tenant, Name: "changed", PolicyResource: policy, Roots: []artifactcohort.RootRef{newArtifact.Root}}}, generation)
	require.NoError(t, err)
	_, _, err = c.ComponentRoots(ctx, pin)
	require.ErrorIs(t, err, contract.ErrConflict, "unprojected publisher generation must fail closed")
	require.NoError(t, c.gate.Recover(ctx))
	currentHash, err := c.AdmitRoot(ctx, root("second"))
	require.NoError(t, err)
	require.NotEqual(t, oldHash, currentHash, "lookup/admission race must be detectable by the caller")
	require.Error(t, c.FenceRoot(ctx, root("second"), oldHash))
	selected = nil
	require.NoError(t, c.gate.Recover(ctx))
	withdrawn, _, err := c.ComponentRoots(ctx, pin)
	require.NoError(t, err)
	require.Empty(t, withdrawn)
	_, err = c.AdmitRoot(ctx, root("second"))
	require.Error(t, err)
}

func BenchmarkComponentRootsPreparedLookup(b *testing.B) {
	for _, count := range []int{1, 100, 10000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			ctx := b.Context()
			publisher := artifactcohort.NewPublisher()
			if _, err := publisher.ReplaceCatalog(ctx, nil, nil, 0); err != nil {
				b.Fatal(err)
			}
			scope := releasevalue.Scope{Tenant: "tenant", Account: "account", Workspace: "workspace", Namespace: "example"}
			pin := releasevalue.ComponentBinding{ComponentKind: "dynamic", ComponentID: "reader-0", ComponentRevision: "1", Method: "GET", Route: "/reader", SchemaFingerprint: strings.Repeat("a", 64), ContentFingerprint: strings.Repeat("b", 64)}
			selections := make([]ActiveSelection, count)
			for i := range selections {
				binding := pin
				binding.ComponentID = fmt.Sprintf("reader-%d", i)
				binding.Root = artifactcohort.RootRef{Kind: "window", URI: fmt.Sprintf("window://example/item-%d", i), Version: 1, Hash: strings.Repeat("c", 64)}
				selections[i].Bindings = []releasevalue.ComponentBinding{binding}
			}
			c := &Controller{config: ControllerConfig{Scope: scope, Publisher: publisher, VerifyScope: func(context.Context) (releasevalue.Scope, error) { return scope, nil }, Authorize: func(context.Context, string) error { return nil }}}
			var err error
			c.gate, err = releasegate.New(&activeGraphStore{}, releasegate.Projection{Reconcile: func(context.Context) error {
				manifest, err := publisher.Snapshot()
				if err == nil {
					c.setActive(manifest, selections)
				}
				return err
			}, Snapshot: func(context.Context) (any, error) { return nil, nil }, Project: func(context.Context, contract.PendingRevision) error { return nil }, Restore: func(context.Context, any) error { return nil }})
			if err != nil {
				b.Fatal(err)
			}
			if err = c.gate.Recover(ctx); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				roots, _, err := c.ComponentRoots(ctx, pin)
				if err != nil || len(roots) != 1 {
					b.Fatalf("exact lookup: %v %v", roots, err)
				}
			}
		})
	}
}
