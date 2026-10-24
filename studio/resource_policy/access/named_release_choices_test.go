package access

import (
	"context"
	"testing"
	"time"

	"github.com/viant/authz"
)

type releaseChoiceStore map[authz.Resource]authz.Document

func (s releaseChoiceStore) Get(_ context.Context, resource authz.Resource) (authz.Document, error) {
	return s[resource], nil
}
func (s releaseChoiceStore) Replace(context.Context, authz.Document, int64, string) (authz.Document, error) {
	return authz.Document{}, authz.ErrDenied
}
func (s releaseChoiceStore) PolicyResources(_ context.Context, tenant string) ([]authz.Resource, error) {
	var result []authz.Resource
	for resource := range s {
		if resource.Tenant == tenant {
			result = append(result, resource)
		}
	}
	return result, nil
}

type releaseChoiceFacts struct{ facts authz.Facts }

func (f releaseChoiceFacts) Resolve(context.Context) (authz.Facts, error) { return f.facts, nil }

func TestReleasePolicyChoicesUseVerifiedAccountAndCompileActions(t *testing.T) {
	allow := authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}
	entry := func(id string, policies map[string]authz.Policy) authz.NamedPolicy {
		return authz.NamedPolicy{Resource: authz.Resource{Tenant: "studio", Kind: "policy", ID: id, Version: "logical"}, Revision: 2, DisplayName: id, Policies: policies}
	}
	listed := []authz.NamedPolicy{
		entry("account/2222/readers", map[string]authz.Policy{"*": allow}),
		entry("account/2222/explicit", map[string]authz.Policy{"describe": allow, "execute": allow}),
		entry("account/2222/denied", map[string]authz.Policy{"*": allow, "execute": {Mode: "deny"}}),
		entry("account/2222/bounded", map[string]authz.Policy{"*": {Mode: "protected", Rule: allow.Rule, EntityType: "advertiser"}}),
		entry("account/2222/discovery-denied", map[string]authz.Policy{"*": allow, "discover": {Mode: "deny"}}),
		entry("account/22220/other", map[string]authz.Policy{"*": allow}),
		entry("account/other/readers", map[string]authz.Policy{"*": allow}),
	}
	choices := releasePolicyChoices(listed, "account/2222/")
	if len(choices) != 2 || choices[0].Resource.ID != "account/2222/readers" || choices[1].Resource.ID != "account/2222/explicit" {
		t.Fatalf("release choices admitted incompatible or other-account policy: %+v", choices)
	}
}

func TestReleaseChoicesExcludeArchivedHeadsFromAuthenticatedInventory(t *testing.T) {
	ctx := context.Background()
	store := releaseChoiceStore{}
	allow := authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}
	for _, item := range []struct {
		id       string
		archived bool
	}{
		{"account/2222/live", false},
		{"account/2222/archived", true},
	} {
		resource := authz.Resource{Tenant: "studio", Kind: "policy", ID: item.id, Version: "logical"}
		store[resource] = authz.Document{Resource: resource, Revision: 1, Policies: map[string]authz.Policy{
			authz.NamedDefinitionAction: {Mode: "definition", Definition: &authz.NamedDefinition{DisplayName: item.id, Archived: item.archived, Policies: map[string]authz.Policy{"*": allow}}},
		}}
	}
	admin := &authz.NamedAdministration{Policies: &authz.Administration{Store: store, Provider: releaseChoiceFacts{facts: authz.Facts{Subject: "editor", Issuer: "issuer", Tenant: "studio", ValidUntil: time.Now().Add(time.Hour)}}}, Inventory: store}
	listed, err := admin.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	choices := releasePolicyChoices(listed, "account/2222/")
	if len(choices) != 1 || choices[0].Resource.ID != "account/2222/live" {
		t.Fatalf("archived policy escaped authenticated inventory: %+v", choices)
	}
}
