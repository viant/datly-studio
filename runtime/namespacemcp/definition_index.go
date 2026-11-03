package namespacemcp

import "sync"

// DefinitionIndex is owned by namespace reconciliation. Readers use the same
// prepared snapshot until a complete configuration refresh replaces it.
type DefinitionIndex struct {
	mu   sync.RWMutex
	byID map[string]Definition
}

// Replace publishes a complete snapshot; caller-owned role slices are copied.
func (i *DefinitionIndex) Replace(rows []Definition) {
	next := make(map[string]Definition, len(rows))
	for _, row := range rows {
		row.AllowedRoles = append([]string(nil), row.AllowedRoles...)
		next[row.NamespaceID] = row
	}
	i.mu.Lock()
	i.byID = next
	i.mu.Unlock()
}

// Lookup returns metadata from the prepared generation. AllowedRoles is immutable.
func (i *DefinitionIndex) Lookup(id string) (Definition, bool) {
	i.mu.RLock()
	row, ok := i.byID[id]
	i.mu.RUnlock()
	return row, ok
}
