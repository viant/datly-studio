package artifactrelease

import (
	"context"
	"sort"

	releasevalue "github.com/viant/datly-studio/sdk/artifactrelease/value"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
)

type componentSourceKey [7]string

func sourceKey(binding releasevalue.ComponentBinding) componentSourceKey {
	return componentSourceKey{binding.ComponentKind, binding.ComponentID, binding.ComponentRevision, binding.Method, binding.Route, binding.SchemaFingerprint, binding.ContentFingerprint}
}

// ComponentRoots returns detached candidate roots for an exact source pin and
// the manifest hash from the same prepared snapshot. Root/DataSourceID on the
// input are not selectors. This is metadata only: callers must AdmitRoot and
// compare/fence the returned hash before using a selected root.
func (c *Controller) ComponentRoots(ctx context.Context, binding releasevalue.ComponentBinding) ([]artifactcohort.RootRef, string, error) {
	if c == nil {
		return nil, "", contract.ErrInvalid
	}
	if err := c.check(ctx, c.config.Scope.Namespace, "use"); err != nil {
		return nil, "", err
	}
	var roots []artifactcohort.RootRef
	var hash string
	err := c.gate.Read(func() error {
		if c.activeGeneration != c.config.Publisher.Generation() {
			return contract.ErrConflict
		}
		roots = append([]artifactcohort.RootRef(nil), c.activeComponents[sourceKey(binding)]...)
		hash = c.active.ManifestHash
		return nil
	})
	return roots, hash, err
}

func indexComponentRoots(selections []ActiveSelection) map[componentSourceKey][]artifactcohort.RootRef {
	unique := map[componentSourceKey]map[artifactcohort.RootRef]struct{}{}
	for _, selection := range selections {
		for _, binding := range selection.Bindings {
			key := sourceKey(binding)
			if unique[key] == nil {
				unique[key] = map[artifactcohort.RootRef]struct{}{}
			}
			unique[key][binding.Root] = struct{}{}
		}
	}
	result := make(map[componentSourceKey][]artifactcohort.RootRef, len(unique))
	for key, set := range unique {
		roots := make([]artifactcohort.RootRef, 0, len(set))
		for root := range set {
			roots = append(roots, root)
		}
		sort.Slice(roots, func(i, j int) bool { return artifactcohort.RootKey(roots[i]) < artifactcohort.RootKey(roots[j]) })
		result[key] = roots
	}
	return result
}
