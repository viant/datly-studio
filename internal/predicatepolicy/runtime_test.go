package predicatepolicy

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/viant/datly-studio/studio/settings"
)

func TestRuntimePolicyTighteningReachesExistingGeneration(t *testing.T) {
	registry := &RuntimePolicies{}
	current := registry.Entry("one", nil)
	other := registry.Entry("two", nil)
	require.True(t, current.AllowsPublic(nil))
	registry.Update("one", 2, &settings.Namespace{Security: &settings.Security{RequireInternalComponents: true}})
	require.False(t, current.AllowsPublic(nil))
	require.True(t, current.AllowsInternal(nil))
	require.False(t, registry.Update("one", 1, &settings.Namespace{}))
	require.False(t, registry.Update("one", 2, &settings.Namespace{}))
	require.False(t, current.AllowsPublic(nil))
	require.True(t, other.AllowsPublic(nil))
	require.Same(t, current, registry.Entry("one", nil))
	require.False(t, registry.Entry("one", nil).AllowsPublic(nil), "stale compilation cannot weaken a committed policy")
	registry.Update("one", 3, &settings.Namespace{Security: &settings.Security{RequireSecurityPredicate: true, PredicatePackages: []string{"example.com/scope"}}})
	require.False(t, current.AllowsPublic(nil))
	require.False(t, current.AllowsInternal(nil))
	require.True(t, current.AllowsPublic(map[string]bool{"example.com/scope": true}))
	registry.Update("one", 4, &settings.Namespace{})
	require.True(t, current.AllowsPublic(nil))
}

func TestConcurrentRuntimePoliciesKeepNewestCommittedRevision(t *testing.T) {
	registry := &RuntimePolicies{}
	policy := registry.Entry("one", nil)
	declared := map[string]bool{"example.com/scope": true}
	start := make(chan struct{})
	var workers sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		workers.Add(1)
		go func(offset int) {
			defer workers.Done()
			<-start
			for revision := offset + 1; revision <= 100; revision += 4 {
				registry.Update("one", int64(revision), &settings.Namespace{Security: &settings.Security{RequireInternalComponents: revision == 100, RequireSecurityPredicate: true, PredicatePackages: []string{"example.com/scope"}}})
				policy.AllowsPublic(declared)
				policy.AllowsInternal(declared)
			}
		}(worker)
	}
	close(start)
	workers.Wait()
	require.False(t, policy.AllowsPublic(declared))
	require.True(t, policy.AllowsInternal(declared))
	require.False(t, policy.AllowsInternal(nil))
}
