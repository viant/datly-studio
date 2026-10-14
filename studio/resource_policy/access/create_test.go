package access

import (
	"context"
	"database/sql"
	"github.com/viant/authz"
	"testing"
	"time"
)

type absentPolicyStore struct{ authz.Store }

func (absentPolicyStore) Get(context.Context, authz.Resource) (authz.Document, error) {
	return authz.Document{}, sql.ErrNoRows
}

type creationFacts struct {
	roles  []string
	tenant string
}

func (p creationFacts) Resolve(context.Context) (authz.Facts, error) {
	return authz.Facts{Subject: "owner", Issuer: "test", Tenant: p.tenant, Roles: p.roles, ValidUntil: time.Now().Add(time.Minute)}, nil
}
func TestInitialPolicyRequiresConfiguredVerifiedRoleAndTenant(t *testing.T) {
	resource := authz.Resource{Kind: "component", ID: "reader", Version: "1", Tenant: "tenant"}
	for _, tc := range []struct {
		name, configured, tenant string
		roles                    []string
		allowed                  bool
	}{
		{"no configured creation authority", "", "tenant", []string{"ROLE_ADMIN"}, false},
		{"reader cannot create", "ROLE_ADMIN", "tenant", []string{"ROLE_READER"}, false},
		{"wrong tenant", "ROLE_ADMIN", "other", []string{"ROLE_ADMIN"}, false},
		{"verified administrator", "ROLE_ADMIN", "tenant", []string{"ROLE_ADMIN"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("STUDIO_ACCESS_POLICY_EDITOR_ROLES", tc.configured)
			service := &authz.Service{Store: absentPolicyStore{}, Provider: creationFacts{roles: tc.roles, tenant: tc.tenant}}
			doc, editor, err := initialPolicy(context.Background(), service, resource)
			if !tc.allowed {
				if err == nil || doc != nil || editor != nil {
					t.Fatal("unauthorized initial policy")
				}
				return
			}
			if err != nil || doc == nil || doc.Revision != 0 || !editor.CanManage || doc.Policies["manageAccess"].Rule.Value != "owner" {
				t.Fatalf("initial policy=%+v context=%+v err=%v", doc, editor, err)
			}
			if _, exists := doc.Policies["execute"]; exists {
				t.Fatal("initial document granted consumption")
			}
		})
	}
}
