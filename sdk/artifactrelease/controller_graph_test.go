package artifactrelease

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/sdk/artifactrelease/releasegate"
	releasevalue "github.com/viant/datly-studio/sdk/artifactrelease/value"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
)

type activeGraphStore struct {
	ReleaseStore // Any unexpected legacy read panics instead of hiding an extra query.
	selection    ActiveSelection
	calls        int
	err          error
}

func (s *activeGraphStore) ActiveSelections(context.Context) ([]releasevalue.ActiveSelection, error) {
	s.calls++
	return []releasevalue.ActiveSelection{s.selection}, s.err
}
func (*activeGraphStore) Pending(context.Context) ([]contract.PendingRevision, error) {
	return nil, nil
}

func TestActiveSnapshotUsesGraphAndPreservesAdmissionFences(t *testing.T) {
	ctx := t.Context()
	scope := releasevalue.Scope{Tenant: "tenant", Account: "account", Workspace: "workspace", Namespace: "example"}
	ref := authz.NamedReference{Resource: authz.Resource{Tenant: scope.Tenant, Kind: "policy", ID: "readers", Version: "logical"}, Revision: 1}
	document := authz.Document{Resource: ref.Resource, Revision: 1, Policies: map[string]authz.Policy{authz.NamedDefinitionAction: {Mode: "definition", Definition: &authz.NamedDefinition{DisplayName: "Readers", Policies: map[string]authz.Policy{"describe": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}, "execute": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}}}}}}
	publisher := artifactcohort.NewPublisher()
	_, err := publisher.ReplaceCatalog(ctx, nil, nil, 0)
	require.NoError(t, err)
	artifact, generation, err := publisher.Publish(ctx, artifactcohort.RootRef{URI: "prompt://example/selected", Kind: "prompt", Version: 1}, json.RawMessage(`{"text":"selected"}`), func(context.Context, json.RawMessage) error { return nil })
	require.NoError(t, err)
	bundle := contract.Bundle{Namespace: scope.Namespace, Roots: []artifactcohort.RootRef{artifact.Root}}
	bundle.Hash = contract.BundleHash(bundle)
	channel := contract.Channel{Key: contract.Key{Tenant: scope.Tenant, Account: scope.Account, Workspace: scope.Workspace, Namespace: scope.Namespace, Name: "current"}, Description: "Selected", Revision: 1, Policies: []authz.NamedReference{ref}, ActiveBundleHash: bundle.Hash}
	compiled, err := contract.CompileCohorts(ctx, channel, bundle, []authz.Document{document})
	require.NoError(t, err)
	_, err = publisher.ReplaceCatalog(ctx, compiled.Policies, compiled.Cohorts, generation)
	require.NoError(t, err)
	selection := ActiveSelection{Channel: channel, Bundle: bundle, Dependencies: compiled.Dependencies}
	for _, tc := range []struct {
		name                    string
		unavailable, staleFence bool
		change                  func(*ActiveSelection)
		readError               bool
		want                    int
	}{
		{name: "admitted", want: 1},
		{name: "unavailable root", unavailable: true},
		{name: "changed policy head", staleFence: true},
		{name: "missing dependencies", change: func(s *ActiveSelection) { s.Dependencies = nil }},
		{name: "unselected manifest", change: func(s *ActiveSelection) { s.Channel.Key.Name = "other" }},
		{name: "graph failure", readError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &activeGraphStore{selection: selection}
			if tc.change != nil {
				tc.change(&store.selection)
			}
			readError := errors.New("graph unavailable")
			if tc.readError {
				store.err = readError
			}
			gate, err := releasegate.New(store, releasegate.Projection{Reconcile: func(context.Context) error { return nil }, Snapshot: func(context.Context) (any, error) { return nil, nil }, Project: func(context.Context, contract.PendingRevision) error { return nil }, Restore: func(context.Context, any) error { return nil }})
			require.NoError(t, err)
			require.NoError(t, gate.Recover(ctx))
			controller := &Controller{gate: gate, config: ControllerConfig{Scope: scope, Store: store, Publisher: publisher,
				VerifyScope: func(context.Context) (releasevalue.Scope, error) { return scope, nil }, Authorize: func(context.Context, string) error { return nil },
				AvailableRoot: func(context.Context, artifactcohort.RootRef) error {
					if tc.unavailable {
						return errors.New("withdrawn")
					}
					return nil
				}, Fence: func(context.Context, []authz.NamedDependency) error {
					if tc.staleFence {
						return errors.New("policy changed")
					}
					return nil
				}}}
			result, err := controller.ActiveSnapshot(ctx)
			require.Equal(t, 1, store.calls)
			if tc.readError {
				require.ErrorIs(t, err, readError)
				return
			}
			require.NoError(t, err)
			require.Len(t, result.Selections, tc.want)
		})
	}
}
