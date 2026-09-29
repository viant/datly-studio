package host

import (
	"context"
	"testing"

	access "github.com/viant/authz"
	dexec "github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
)

func TestCatalogPoliciesRetainExactSourceAndInheritedSkillOwnership(t *testing.T) {
	role := &access.Rule{Kind: "role", Value: "reader"}
	otherRole := &access.Rule{Kind: "role", Value: "other"}
	owner := access.Resource{Kind: "component", ID: "owner", Version: "4", Tenant: "one"}
	later := access.Resource{Kind: "component", ID: "owner", Version: "5", Tenant: "one"}
	private := access.Resource{Kind: "component", ID: "private", Version: "4", Tenant: "one"}
	policies := func(rule *access.Rule) map[string]access.Policy {
		return map[string]access.Policy{"discover": {Mode: "protected", Rule: rule}, "describe": {Mode: "protected", Rule: rule}}
	}
	docs := map[access.Resource]access.Document{owner: {Resource: owner, Revision: 1, Policies: policies(role)}, later: {Resource: later, Revision: 1, Policies: policies(otherRole)}, private: {Resource: private, Revision: 1, Policies: policies(otherRole)}}
	service := &Service{config: Config{Access: &ResourceAccessConfig{Tenant: "one"}}, resourceAccess: &access.Service{Store: resourceStore{documents: docs}, Provider: runtimeFacts{}}}
	key := spec.Key{Kind: "reader", Scope: "fixture", Name: "owner"}
	components := map[spec.Key]string{key: "owner"}
	versions := map[string]int{"owner": 4, "private": 4}
	folders := map[string]string{"skill://guide/": "owner", "skill://guide/private/": "private"}
	tool, resource, read := service.catalogAuthorizers(components, versions, folders)
	// Publication inputs cannot rewrite a retained generation's ACL identity.
	components[key] = "private"
	versions["owner"] = 5
	folders["skill://guide/"] = "private"
	for _, action := range []string{"discover", "describe"} {
		if err := tool(context.Background(), dexec.ComponentTarget{Component: key}, action); err != nil {
			t.Fatal(err)
		}
		if err := resource(context.Background(), "skill://guide/SKILL.md", action); err != nil {
			t.Fatal(err)
		}
	}
	if err := read(context.Background(), "skill://guide/SKILL.md"); err != nil {
		t.Fatal(err)
	}
	for _, uri := range []string{"skill://guide/private/SKILL.md", "skill://unknown/SKILL.md", "skill://guide/%2e%2e/private/SKILL.md"} {
		if err := read(context.Background(), uri); err == nil {
			t.Fatalf("unowned or denied file permitted: %s", uri)
		}
	}
	if err := tool(context.Background(), dexec.ComponentTarget{}, "discover"); err == nil {
		t.Fatal("tool without a server-owned source identity permitted")
	}
}

func TestExplicitSkillBindingUsesTheRequestedCatalogAction(t *testing.T) {
	skill := access.Resource{Kind: "skill", ID: "guide", Version: "1", Tenant: "one"}
	docs := map[access.Resource]access.Document{skill: {Resource: skill, Revision: 1, Policies: map[string]access.Policy{"retrieve": {Mode: "protected", Rule: &access.Rule{Kind: "role", Value: "reader"}}, "discover": {Mode: "protected", Rule: &access.Rule{Kind: "role", Value: "other"}}}}}
	service := &Service{config: Config{Access: &ResourceAccessConfig{Tenant: "one", ResourceBindings: map[string]access.Resource{"skill://guide/": skill}}}, resourceAccess: &access.Service{Store: resourceStore{documents: docs}, Provider: runtimeFacts{}}}
	_, resource, read := service.catalogAuthorizers(nil, map[string]int{"owner": 4}, map[string]string{"skill://guide/": "owner"})
	if err := read(context.Background(), "skill://guide/SKILL.md"); err != nil {
		t.Fatal(err)
	}
	if err := resource(context.Background(), "skill://guide/SKILL.md", "discover"); err == nil {
		t.Fatal("retrieve permission silently granted catalog discovery")
	}
}
