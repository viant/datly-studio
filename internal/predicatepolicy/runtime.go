package predicatepolicy

import (
	"sync"
	"sync/atomic"

	"github.com/viant/datly-studio/studio/settings"
)

// RuntimePolicies shares committed policy changes with existing and staged
// generations. Request callbacks retain one namespace entry, not a datastore.
type RuntimePolicies struct {
	mu      sync.Mutex
	entries map[string]*RuntimePolicy
}

type RuntimePolicy struct {
	current atomic.Pointer[runtimeSecurity]
}
type runtimeSecurity struct {
	revision int64
	internal bool
}

func securitySnapshot(policy *settings.Namespace) *runtimeSecurity {
	result := &runtimeSecurity{}
	if policy != nil && policy.Security != nil {
		result.internal = policy.Security.RequireInternalComponents
	}
	return result
}

// Entry never replaces a newer committed policy with a compiler's older snapshot.
func (r *RuntimePolicies) Entry(namespaceID string, initial *settings.Namespace) *RuntimePolicy {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.entries == nil {
		r.entries = map[string]*RuntimePolicy{}
	}
	entry := r.entries[namespaceID]
	if entry == nil {
		entry = &RuntimePolicy{}
		entry.current.Store(securitySnapshot(initial))
		r.entries[namespaceID] = entry
	}
	return entry
}

func (r *RuntimePolicies) Update(namespaceID string, revision int64, policy *settings.Namespace) bool {
	entry := r.Entry(namespaceID, policy)
	next := securitySnapshot(policy)
	next.revision = revision
	for {
		previous := entry.current.Load()
		if previous != nil && revision <= previous.revision {
			return false
		}
		if entry.current.CompareAndSwap(previous, next) {
			return true
		}
	}
}

// AllowsPublic adds only local policy checks to the existing transport gate.
// Trusted internal invocation has its own native contract and authorization.
func (p *RuntimePolicy) AllowsPublic(declared map[string]bool) bool {
	return p.allows(declared, false)
}

func (p *RuntimePolicy) AllowsInternal(declared map[string]bool) bool {
	return p.allows(declared, true)
}

func (p *RuntimePolicy) allows(declared map[string]bool, internal bool) bool {
	policy := p.current.Load()
	if policy == nil || policy.internal && !internal {
		return false
	}
	// Declaration requirements belong to authoring validation, not live access.
	return true
}
