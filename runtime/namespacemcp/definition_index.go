package namespacemcp

import (
	"context"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"sync"
)

// DefinitionIndex is owned by namespace reconciliation. Readers use the same
// prepared snapshot until a complete configuration refresh replaces it.
type DefinitionIndex struct {
	mu       sync.RWMutex
	byID     map[string]Definition
	byName   map[string][]Definition
	rows     []Definition
	policies map[string]namespaceaccess.Policy
}

// Replace publishes a complete snapshot; caller-owned role slices are copied.
func (i *DefinitionIndex) Replace(rows []Definition) {
	next := make(map[string]Definition, len(rows))
	names := make(map[string][]Definition)
	policies := make(map[string]namespaceaccess.Policy, len(rows))
	snapshot := make([]Definition, 0, len(rows))
	for _, row := range rows {
		row.AllowedRoles = append([]string(nil), row.AllowedRoles...)
		next[row.NamespaceID] = row
		policies[row.NamespaceID] = (namespaceaccess.Policy{ID: row.NamespaceID, OwnerID: row.OwnerID, Visibility: row.Visibility, Roles: row.AllowedRoles}).Prepare()
		names[row.Name] = append(names[row.Name], row)
		snapshot = append(snapshot, row)
	}
	i.mu.Lock()
	i.byID = next
	i.policies = policies
	i.byName = names
	i.rows = snapshot
	i.mu.Unlock()
}

// Lookup returns metadata from the prepared generation. AllowedRoles is immutable.
func (i *DefinitionIndex) Lookup(id string) (Definition, bool) {
	i.mu.RLock()
	row, ok := i.byID[id]
	i.mu.RUnlock()
	return row, ok
}

// LookupName retains duplicate owner/name candidates for the host's existing
// ambiguity rules. Returned metadata is immutable.
func (i *DefinitionIndex) LookupName(name string) []Definition {
	i.mu.RLock()
	rows := i.byName[name]
	i.mu.RUnlock()
	return rows
}

// Load exposes the prepared, immutable snapshot to existing control-plane readers.
func (i *DefinitionIndex) Load(ctx context.Context) ([]Definition, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	i.mu.RLock()
	rows := i.rows
	i.mu.RUnlock()
	return rows, nil
}

// LookupAccess reads the metadata and compiled namespace policy atomically.
func (i *DefinitionIndex) LookupAccess(id string) (Definition, namespaceaccess.Policy, bool) {
	i.mu.RLock()
	row, ok := i.byID[id]
	policy := i.policies[id]
	i.mu.RUnlock()
	return row, policy, ok
}

// Update refreshes one namespace after its control-plane storage read. Other
// namespaces remain untouched; request paths never call this method.
func (i *DefinitionIndex) Update(row Definition) {
	row.AllowedRoles = append([]string(nil), row.AllowedRoles...)
	policy := (namespaceaccess.Policy{ID: row.NamespaceID, OwnerID: row.OwnerID, Visibility: row.Visibility, Roles: row.AllowedRoles}).Prepare()
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.byID == nil {
		i.byID = map[string]Definition{}
		i.byName = map[string][]Definition{}
		i.policies = map[string]namespaceaccess.Policy{}
	}
	previous, existed := i.byID[row.NamespaceID]
	if existed {
		values := i.byName[previous.Name]
		kept := make([]Definition, 0, len(values))
		for _, value := range values {
			if value.NamespaceID != row.NamespaceID {
				kept = append(kept, value)
			}
		}
		if len(kept) == 0 {
			delete(i.byName, previous.Name)
		} else {
			i.byName[previous.Name] = kept
		}
	}
	i.byID[row.NamespaceID] = row
	i.policies[row.NamespaceID] = policy
	i.byName[row.Name] = append(append([]Definition(nil), i.byName[row.Name]...), row)
	rows := make([]Definition, 0, len(i.byID))
	for _, value := range i.byID {
		rows = append(rows, value)
	}
	i.rows = rows
}
