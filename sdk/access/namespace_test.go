package access

import (
	"context"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/sdk"
	catalog "github.com/viant/datly-studio/studio/resource_policy/catalog"
	"strings"
	"testing"
	"time"
)

type componentScopeTransport struct{ calls int }

func (t *componentScopeTransport) Invoke(_ context.Context, _ string, input, output any) error {
	t.calls++
	id := input.(map[string]any)["id"].(string)
	output.(*sdk.Component).ID = id
	return nil
}

func TestPolicyManagementNamespaceBoundary(t *testing.T) {
	a, b := strings.Repeat("a", 64), strings.Repeat("b", 64)
	own := authz.Resource{Kind: "component", ID: "alpha", Version: "1", Tenant: "tenant"}
	foreign := authz.Resource{Kind: "component", ID: "beta", Version: "1", Tenant: "tenant"}
	skill := authz.Resource{Kind: "skill", ID: "beta-skill", Version: "1", Tenant: "tenant"}
	fixture := &catalogFixture{rows: []*catalog.Entry{
		{Kind: own.Kind, ID: own.ID, Version: "1", ComponentID: "alpha", NamespaceID: a},
		{Kind: foreign.Kind, ID: foreign.ID, Version: "1", ComponentID: "beta", NamespaceID: b},
		{Kind: skill.Kind, ID: skill.ID, Version: "1", ComponentID: "beta", NamespaceID: b},
	}, documents: map[authz.Resource]authz.Document{}}
	for _, r := range []authz.Resource{own, foreign, skill} {
		fixture.documents[r] = authz.Document{Resource: r, Revision: 1, Policies: map[string]authz.Policy{
			"viewAccess":   {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: "owner"}},
			"manageAccess": {Mode: "protected", Rule: &authz.Rule{Kind: "subject", Value: "owner"}},
		}}
	}
	next := &componentScopeTransport{}
	transport := &Transport{Next: next, Service: &authz.Service{Store: fixture, Provider: catalogTestProvider{authz.Facts{Subject: "owner", Tenant: "tenant", Issuer: "test", ValidUntil: time.Now().Add(time.Hour)}}}, Catalog: &Catalog{Source: fixture}}
	ctx := sdk.WithNamespaceSelection(context.Background(), a)
	for _, r := range []authz.Resource{foreign, skill} {
		for _, op := range []string{OperationGet, OperationContext, OperationReplace} {
			var input any = r
			var output any = &authz.Document{}
			if op == OperationReplace {
				input = fixture.documents[r]
			}
			if op == OperationContext {
				output = &authz.EditorContext{}
			}
			if err := transport.Invoke(ctx, op, input, output); err == nil {
				t.Fatalf("%s escaped selected namespace for %s", op, r.Kind)
			}
		}
	}
	if next.calls != 0 {
		t.Fatal("foreign resource reached component lookup")
	}
	var doc authz.Document
	if err := transport.Invoke(ctx, OperationGet, own, &doc); err != nil || doc.Resource != own {
		t.Fatalf("local policy unavailable: %+v %v", doc, err)
	}
	transport.Catalog.Service = transport.Service
	for _, row := range fixture.rows {
		row.OwnerID = "owner"
	}
	var page CatalogPage
	if err := transport.Invoke(ctx, OperationList, CatalogInput{Limit: 1}, &page); err != nil || len(page.Items) != 1 || page.Items[0].ID != "alpha" || page.HasMore {
		t.Fatalf("authenticated catalog wrapper escaped namespace: %+v %v", page, err)
	}
	for _, r := range []authz.Resource{{Kind: "component", ID: "missing", Version: "1"}, {Kind: "component", ID: "alpha", Version: "2"}, {Kind: "other", ID: "alpha", Version: "1"}} {
		if err := CheckNamespaceResource(ctx, &a, r, fixture, func(context.Context, string) bool { return true }); err == nil {
			t.Fatalf("unknown resource accepted: %+v", r)
		}
	}
	if err := CheckNamespaceResource(ctx, &a, own, fixture, func(context.Context, string) bool { return false }); err == nil {
		t.Fatal("component visibility bypassed")
	}
	invalid := "invalid"
	if err := CheckNamespaceResource(ctx, &invalid, own, fixture, func(context.Context, string) bool { return true }); err == nil {
		t.Fatal("invalid selection accepted")
	}
}
