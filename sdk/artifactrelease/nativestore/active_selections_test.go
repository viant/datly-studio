package nativestore

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/authz"
	releasevalue "github.com/viant/datly-studio/sdk/artifactrelease/value"
	snapshotview "github.com/viant/datly-studio/studio/artifact_release/snapshot"
	"github.com/viant/datly/exec"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
)

type selectionInvoker func(context.Context, exec.ComponentRequest) (any, error)

func (f selectionInvoker) InvokeComponent(ctx context.Context, request exec.ComponentRequest) (any, error) {
	return f(ctx, request)
}

func TestActiveSelectionsUsesOneGraphAndRejectsBrokenRelations(t *testing.T) {
	scope := releasevalue.Scope{Tenant: "tenant", Account: "account", Workspace: "workspace", Namespace: "example"}
	text := func(s string) *string { return &s }
	bundle := contract.Bundle{Namespace: scope.Namespace, Roots: []artifactcohort.RootRef{{URI: "prompt://example/selected", Kind: "prompt", Version: 1, Hash: strings.Repeat("a", 64)}}}
	bundle.Hash = contract.BundleHash(bundle)
	require.NoError(t, contract.ValidateBundle(bundle))
	body, err := json.Marshal(bundle)
	require.NoError(t, err)
	_, bindingRaw, bindingHash, err := releasevalue.ValidateBindings(bundle, nil)
	require.NoError(t, err)
	policies, err := json.Marshal([]authz.NamedReference{{Resource: authz.Resource{Tenant: scope.Tenant, Kind: "policy", ID: "readers", Version: "logical"}, Revision: 1}})
	require.NoError(t, err)
	raw := func(v []byte) *json.RawMessage { r := json.RawMessage(v); return &r }
	row := func(name string) *snapshotview.ReleaseChannel {
		return &snapshotview.ReleaseChannel{ScopeId: text(scope.ID()), TenantId: text(scope.Tenant), AccountId: text(scope.Account), WorkspaceId: text(scope.Workspace), Namespace: text(scope.Namespace), Name: text(name), Description: text("Selected"), Revision: 1, ActiveBundleHash: text(bundle.Hash), PolicyRefsJson: raw(policies),
			Bundle:  &snapshotview.StoredBundle{ScopeId: text(scope.ID()), TenantId: text(scope.Tenant), AccountId: text(scope.Account), WorkspaceId: text(scope.Workspace), Namespace: text(scope.Namespace), BundleHash: text(bundle.Hash), BundleJson: raw(body)},
			Binding: &snapshotview.BundleBinding{ScopeId: text(scope.ID()), BundleHash: text(bundle.Hash), BindingsJson: raw(bindingRaw), BindingsHash: text(bindingHash)}}
	}
	for _, tc := range []struct {
		name   string
		change func(*snapshotview.ReleaseChannel)
	}{{name: "complete graph"}, {name: "foreign child", change: func(r *snapshotview.ReleaseChannel) { r.Bundle.AccountId = text("foreign") }}, {name: "missing child", change: func(r *snapshotview.ReleaseChannel) { r.Binding = nil }}, {name: "changed bundle", change: func(r *snapshotview.ReleaseChannel) { r.Bundle.BundleJson = raw([]byte(`{}`)) }}} {
		t.Run(tc.name, func(t *testing.T) {
			rows := []*snapshotview.ReleaseChannel{row("current"), row("next")}
			if tc.change != nil {
				tc.change(rows[1])
			}
			calls := 0
			store, err := New(selectionInvoker(func(_ context.Context, request exec.ComponentRequest) (any, error) {
				calls++
				require.Equal(t, "snapshot", request.Target.Component.Name)
				input := request.Input.(*snapshotview.Input)
				require.Equal(t, scope.ID(), input.ScopeId)
				require.True(t, input.ActiveOnly)
				require.True(t, input.Has.ActiveOnly)
				require.False(t, input.Has.Name)
				return &snapshotview.Output{Data: rows}, nil
			}), scope, func(context.Context) (string, error) { return "actor", nil })
			require.NoError(t, err)
			result, err := store.ActiveSelections(t.Context())
			require.Equal(t, 1, calls)
			if tc.change != nil {
				require.ErrorIs(t, err, contract.ErrInvalid)
				require.Nil(t, result, "a broken graph must not return partial selections")
				return
			}
			require.NoError(t, err)
			require.Len(t, result, 2)
			require.Equal(t, "current", result[0].Channel.Key.Name)
			require.Equal(t, "next", result[1].Channel.Key.Name)
			require.Equal(t, bundle, result[0].Bundle)
		})
	}
}
