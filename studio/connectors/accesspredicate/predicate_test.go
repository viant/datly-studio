package accesspredicate

import (
	"context"
	"strings"
	"testing"
)

type scope struct{ subject, name, permission string }

func (value scope) ConnectorAccessScope() (string, string, string) {
	return value.subject, value.name, value.permission
}

func TestConnectorAccessUsesDeclaredPermissionColumns(t *testing.T) {
	for _, test := range []struct{ permission, column string }{
		{"view", "can_view"}, {"run", "can_run"}, {"edit", "can_edit"},
		{"publish", "can_publish"}, {"dql", "can_use_dql"},
	} {
		criteria, err := (&ConnectorAccess{Input: scope{subject: "alice", name: "main", permission: test.permission}}).Compute(context.Background(), nil)
		if err != nil || !strings.Contains(criteria.Expression, test.column) || len(criteria.Placeholders) != 3 {
			t.Fatalf("%s criteria=%+v err=%v", test.permission, criteria, err)
		}
	}
	if _, err := (&ConnectorAccess{Input: scope{name: "main", permission: "view"}}).Compute(context.Background(), nil); err == nil {
		t.Fatal("empty subject accepted")
	}
}
