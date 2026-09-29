package store_write

import (
	"context"
	"github.com/viant/datly-studio/internal/namespaceaccess"
	"github.com/viant/xdatly/handler"
	"testing"
)

func TestACLNamespaceOwnershipCannotBeOmittedOrMoved(t *testing.T) {
	rules := &ACLStoreRules{}
	id := namespaceaccess.ID("owner", "alpha")
	for _, row := range []*StoredACL{
		{Has: &StoredACLHas{}},
		{NamespaceId: "invalid", Has: &StoredACLHas{NamespaceId: true}},
		{NamespaceId: id, Has: &StoredACLHas{}},
	} {
		if err := rules.Init(context.Background(), row, handler.LifecycleContext[StoredACL, handler.NoParent, Output]{}); err == nil {
			t.Fatal("ACL mutation accepted missing ownership")
		}
	}
	previous := &StoredACL{NamespaceId: id}
	moved := &StoredACL{NamespaceId: namespaceaccess.ID("owner", "beta"), ShouldDelete: true, Has: &StoredACLHas{NamespaceId: true}}
	state := handler.LifecycleContext[StoredACL, handler.NoParent, Output]{}
	state.Previous = previous
	if err := rules.Init(context.Background(), moved, state); err == nil {
		t.Fatal("ACL ownership change accepted")
	}
}
