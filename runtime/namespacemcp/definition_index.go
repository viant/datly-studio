package namespacemcp

import (
	"context"
	"sync"
)

// DefinitionIndex is owned by namespace reconciliation. Readers use the same
// prepared snapshot until a complete configuration refresh replaces it.
type DefinitionIndex struct {
	mu     sync.RWMutex
	byID   map[string]Definition
	byName map[string][]Definition
	rows   []Definition
}

// Replace publishes a complete snapshot; caller-owned role slices are copied.
func (i *DefinitionIndex) Replace(rows []Definition) {
	next := make(map[string]Definition, len(rows))
	names := make(map[string][]Definition)
	snapshot := make([]Definition, 0, len(rows))
	for _, row := range rows {
		row.AllowedRoles = append([]string(nil), row.AllowedRoles...)
		next[row.NamespaceID] = row
		names[row.Name] = append(names[row.Name], row)
		snapshot = append(snapshot, row)
	}
	i.mu.Lock()
	i.byID = next
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
