package namespacemcp

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Change struct {
	NamespaceID string    `json:"namespaceId"`
	Status      string    `json:"status"`
	Endpoint    *Endpoint `json:"endpoint,omitempty"`
	Message     string    `json:"message,omitempty"`
	Cause       error     `json:"-"`
}

type Reconciler struct {
	Manager     *Manager
	Definitions Definitions
	mu          sync.Mutex
}

// Apply reads a complete persisted snapshot before changing any listeners.
// One listener failure does not stop reconciliation of independent namespaces.
func (r *Reconciler) Apply(ctx context.Context) ([]Change, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Manager == nil || r.Definitions == nil {
		return nil, fmt.Errorf("namespace manager and definitions are required")
	}
	definitions, err := r.Definitions.Load(ctx)
	if err != nil {
		return nil, err
	}
	desired := map[string]Definition{}
	for _, definition := range definitions {
		decoded, err := hex.DecodeString(definition.NamespaceID)
		if err != nil || len(decoded) != 32 || strings.ToLower(definition.NamespaceID) != definition.NamespaceID || definition.Port < 0 || definition.Port > 65535 {
			return nil, fmt.Errorf("invalid persisted namespace MCP configuration")
		}
		if _, exists := desired[definition.NamespaceID]; exists {
			return nil, fmt.Errorf("duplicate persisted namespace identity")
		}
		desired[definition.NamespaceID] = definition
	}
	r.Manager.mu.Lock()
	current := map[string]int{}
	for id, entry := range r.Manager.entries {
		current[id] = entry.requestedPort
	}
	r.Manager.mu.Unlock()
	ids := map[string]bool{}
	for id := range desired {
		ids[id] = true
	}
	for id := range current {
		ids[id] = true
	}
	ordered := make([]string, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Strings(ordered)
	var changes []Change
	var failures error
	for _, id := range ordered {
		definition, present := desired[id]
		oldPort, running := current[id]
		change := Change{NamespaceID: id}
		if !present || !definition.Enabled {
			if !running {
				continue
			}
			err = r.Manager.Stop(ctx, id)
			change.Status = "stopped"
		} else {
			var endpoint Endpoint
			if running && oldPort != definition.Port {
				endpoint, err = r.Manager.Rebind(ctx, id, definition.Port)
				change.Status = "rebound"
			} else {
				endpoint, err = r.Manager.Start(ctx, id, definition.Port)
				change.Status = "ready"
			}
			if endpoint.NamespaceID != "" {
				change.Endpoint = &endpoint
			}
		}
		if err != nil {
			change.Status = "failed"
			change.Message = "Namespace MCP endpoint could not be updated"
			change.Cause = err
			if endpoint, ok := r.Manager.Get(id); ok {
				change.Endpoint = &endpoint
			}
			failures = errors.Join(failures, fmt.Errorf("namespace %s: %w", id, err))
		}
		changes = append(changes, change)
	}
	return changes, failures
}
