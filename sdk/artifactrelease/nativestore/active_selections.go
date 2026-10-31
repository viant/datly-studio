package nativestore

import (
	"context"
	"reflect"

	releasevalue "github.com/viant/datly-studio/sdk/artifactrelease/value"
	snapshotview "github.com/viant/datly-studio/studio/artifact_release/snapshot"
	contract "github.com/viant/forge/backend/artifactrelease"
)

// ActiveSelections reads the complete committed metadata graph in one native
// invocation. Datly owns relation batching; this adapter only validates and
// decodes the already-related values into the controller contract.
func (s *Store) ActiveSelections(ctx context.Context) ([]releasevalue.ActiveSelection, error) {
	return s.channelSelections(ctx, true)
}

// ChannelSelections includes unpublished channels and their optional committed
// metadata in the same native snapshot graph used by ActiveSelections.
func (s *Store) ChannelSelections(ctx context.Context) ([]releasevalue.ActiveSelection, error) {
	return s.channelSelections(ctx, false)
}

func (s *Store) channelSelections(ctx context.Context, activeOnly bool) ([]releasevalue.ActiveSelection, error) {
	if !s.ready(ctx) {
		return nil, contract.ErrInvalid
	}
	input := new(snapshotview.Input)
	input.SetScopeId(s.scope.ID())
	if activeOnly {
		input.SetActiveOnly(true)
	}
	value, err := s.invoke(ctx, reflect.TypeFor[snapshotview.SnapshotComponent](), "snapshot", "GET", input)
	if err != nil {
		return nil, err
	}
	output, ok := value.(*snapshotview.Output)
	if !ok || output == nil {
		return nil, contract.ErrInvalid
	}
	result := make([]releasevalue.ActiveSelection, 0, len(output.Data))
	for _, row := range output.Data {
		if row == nil || str(row.ScopeId) != s.scope.ID() {
			return nil, contract.ErrInvalid
		}
		selected, err := channel(row)
		if err != nil || !s.scope.Owns(selected.Key) {
			return nil, contract.ErrInvalid
		}
		if selected.ActiveBundleHash == "" {
			if activeOnly || row.Bundle != nil || row.Binding != nil {
				return nil, contract.ErrInvalid
			}
			result = append(result, releasevalue.ActiveSelection{Channel: selected})
			continue
		}
		bundleRow, bindingRow := row.Bundle, row.Binding
		if bundleRow == nil || bundleRow.BundleJson == nil || bindingRow == nil || bindingRow.BindingsJson == nil ||
			str(bundleRow.ScopeId) != s.scope.ID() || str(bundleRow.BundleHash) != selected.ActiveBundleHash ||
			str(bundleRow.TenantId) != s.scope.Tenant || str(bundleRow.AccountId) != s.scope.Account || str(bundleRow.WorkspaceId) != s.scope.Workspace || str(bundleRow.Namespace) != s.scope.Namespace ||
			str(bindingRow.ScopeId) != s.scope.ID() || str(bindingRow.BundleHash) != selected.ActiveBundleHash {
			return nil, contract.ErrInvalid
		}
		var bundle contract.Bundle
		if releasevalue.StrictJSON(*bundleRow.BundleJson, &bundle) != nil || contract.ValidateBundle(bundle) != nil || bundle.Namespace != s.scope.Namespace || bundle.Hash != selected.ActiveBundleHash {
			return nil, contract.ErrInvalid
		}
		bindings, err := releasevalue.DecodeBindings(bundle, *bindingRow.BindingsJson, str(bindingRow.BindingsHash))
		if err != nil {
			return nil, err
		}
		var dependenciesRaw []byte
		if row.ActiveDependenciesJson != nil {
			dependenciesRaw = *row.ActiveDependenciesJson
		}
		dependencies, err := releasevalue.DecodeDependencies(dependenciesRaw)
		if err != nil {
			return nil, err
		}
		result = append(result, releasevalue.ActiveSelection{Channel: selected, Bundle: bundle, Bindings: bindings, Dependencies: dependencies})
	}
	return result, nil
}
