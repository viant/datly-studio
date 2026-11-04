package namespaceaccess

import (
	"github.com/viant/authz"
	"testing"
	"time"
)

func TestPreparedNamespaceRolesPreserveAccess(t *testing.T) {
	now := time.Now()
	for _, visibility := range []string{"", Private, Public, "invalid"} {
		for _, owner := range []string{"", "owner"} {
			for _, roles := range [][]string{nil, {"viewer"}, {"", " viewer", "viewer ", "viewer", "viewer"}} {
				policy := Policy{ID: "namespace", OwnerID: owner, Visibility: visibility, Roles: roles}
				prepared := policy.Prepare()
				for _, facts := range []authz.Facts{{}, {Subject: "owner", Issuer: "fixture", ValidUntil: now.Add(time.Minute)}, {Subject: "reader", Issuer: "fixture", ValidUntil: now.Add(time.Minute), Roles: []string{"viewer"}}, {Subject: "reader", Issuer: "fixture", ValidUntil: now.Add(time.Minute), Roles: []string{"other"}}, {Subject: "reader", Issuer: "fixture", ValidUntil: now.Add(-time.Minute), Roles: []string{"viewer"}}} {
					if got, want := prepared.CanView(facts, now), policy.CanView(facts, now); got != want {
						t.Fatalf("prepared policy changed access: visibility=%q owner=%q roles=%v", visibility, owner, roles)
					}
				}
			}
		}
	}
}
