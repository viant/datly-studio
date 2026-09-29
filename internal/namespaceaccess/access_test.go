package namespaceaccess

import (
	"github.com/viant/authz"
	"testing"
	"time"
)

func TestNamespaceVisibilityAndIndependentScope(t *testing.T) {
	now := time.Now()
	policy := Policy{ID: ID("owner", "finance"), OwnerID: "owner", Visibility: Private, Roles: []string{"analyst"}}
	owner := authz.Facts{Subject: "owner", Issuer: "verified", ValidUntil: now.Add(time.Minute)}
	analyst := authz.Facts{Subject: "viewer", Issuer: "verified", ValidUntil: now.Add(time.Minute), Roles: []string{"analyst"}}
	if !policy.CanView(owner, now) || !policy.CanManage(owner, now) || !policy.CanView(analyst, now) || policy.CanManage(analyst, now) {
		t.Fatal("owner/role boundary incorrect")
	}
	analyst.Roles = nil
	if policy.CanView(analyst, now) {
		t.Fatal("unassigned caller saw private namespace")
	}
	policy.Visibility = Public
	if !policy.CanView(authz.Facts{}, now) || policy.CanManage(authz.Facts{}, now) {
		t.Fatal("public visibility granted management")
	}
	policy.Visibility = Private
	owner.ValidUntil = now
	if policy.CanView(owner, now) {
		t.Fatal("expired identity authorized")
	}
	if Contains(policy.ID, ID("owner", "sales")) || Contains("", policy.ID) || !Contains(policy.ID, policy.ID) {
		t.Fatal("current namespace isolation failed")
	}
	if ID("alice", "same") == ID("bob", "same") {
		t.Fatal("owners share namespace identity")
	}
}
