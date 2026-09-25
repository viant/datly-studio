package accesspredicate

import (
	"context"
	"strings"
	"testing"
)

type scope struct{ subject, name, permission string }

func (value scope) NamespaceAccessScope() (string, string, string) {
	return value.subject, value.name, value.permission
}

func TestNamespaceAccessPermissionModes(t *testing.T) {
	for _, test := range []struct {
		permission, column string
		ownerOnly          bool
	}{
		{"view", "can_view", false}, {"run", "can_run", false}, {"dql", "can_use_dql", false},
		{"edit", "", true}, {"publish", "", true},
	} {
		criteria, err := (&NamespaceAccess{Input: scope{subject: "alice", name: "finance", permission: test.permission}}).Compute(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if test.ownerOnly {
			if strings.Contains(criteria.Expression, "report_acl") || len(criteria.Placeholders) != 2 {
				t.Fatalf("%s criteria=%+v", test.permission, criteria)
			}
		} else if !strings.Contains(criteria.Expression, test.column) || len(criteria.Placeholders) != 3 {
			t.Fatalf("%s criteria=%+v", test.permission, criteria)
		}
	}
	if _, err := (&NamespaceAccess{Input: scope{name: "finance", permission: "view"}}).Compute(context.Background(), nil); err == nil {
		t.Fatal("empty subject accepted")
	}
}
