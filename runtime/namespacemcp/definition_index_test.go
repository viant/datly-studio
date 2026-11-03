package namespacemcp

import (
	"context"
	"strings"
	"testing"
)

func TestDefinitionIndexPublishesRoleChangesAndWithdrawal(t *testing.T) {
	index := &DefinitionIndex{}
	rows := []Definition{{NamespaceID: "one", Active: true, AllowedRoles: []string{"viewer"}}}
	index.Replace(rows)
	rows[0].AllowedRoles[0] = "mutated"
	if row, ok := index.Lookup("one"); !ok || row.AllowedRoles[0] != "viewer" {
		t.Fatal("snapshot borrowed mutable source roles")
	}
	index.Replace([]Definition{{NamespaceID: "one", Active: true, AllowedRoles: []string{"manager"}}})
	if row, _ := index.Lookup("one"); row.AllowedRoles[0] != "manager" {
		t.Fatal("role change not published")
	}
	index.Replace(nil)
	if _, ok := index.Lookup("one"); ok {
		t.Fatal("withdrawn namespace retained")
	}
}

func BenchmarkDefinitionIndexLookup(b *testing.B) {
	index := &DefinitionIndex{}
	index.Replace([]Definition{{NamespaceID: "one", Active: true, AllowedRoles: []string{"viewer"}}})
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if _, ok := index.Lookup("one"); !ok {
			b.Fatal("missing namespace")
		}
	}
}

func TestReconciliationDoesNotReplaceNewerNamespaceMetadata(t *testing.T) {
	id := strings.Repeat("a", 64)
	index := &DefinitionIndex{}
	source := &controlledStaleDefinitions{snapshot: []Definition{{NamespaceID: id, Active: true, AllowedRoles: []string{"old"}}}, captured: make(chan struct{}), release: make(chan struct{})}
	manager := &Manager{entries: map[string]*instance{}, configurationRevisions: map[string]uint64{}}
	reconciler := &Reconciler{Manager: manager, Definitions: source, Index: index}
	done := make(chan error, 1)
	go func() { _, err := reconciler.Apply(context.Background()); done <- err }()
	<-source.captured
	manager.mu.Lock()
	manager.configurationRevisions[id]++
	index.Replace([]Definition{{NamespaceID: id, Active: true, AllowedRoles: []string{"current"}}})
	manager.mu.Unlock()
	close(source.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	row, ok := index.Lookup(id)
	if !ok || len(row.AllowedRoles) != 1 || row.AllowedRoles[0] != "current" {
		t.Fatal("older read restored stale namespace permissions")
	}
}

func TestDefinitionIndexNameLookupPreservesOwnerCandidates(t *testing.T) {
	index := &DefinitionIndex{}
	index.Replace([]Definition{{NamespaceID: "one", Name: "team", OwnerID: "first"}, {NamespaceID: "two", Name: "team", OwnerID: "second"}, {NamespaceID: "three", Name: "other", OwnerID: "first"}})
	candidates := index.LookupName("team")
	if len(candidates) != 2 || candidates[0].OwnerID != "first" || candidates[1].OwnerID != "second" {
		t.Fatal("name lookup lost ambiguity candidates")
	}
	if len(index.LookupName("missing")) != 0 {
		t.Fatal("unknown name returned a candidate")
	}
	index.Replace([]Definition{{NamespaceID: "one", Name: "renamed", OwnerID: "first"}})
	if len(index.LookupName("team")) != 0 || len(index.LookupName("renamed")) != 1 {
		t.Fatal("name index retained an old generation")
	}
}
