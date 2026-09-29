package access

import (
	"context"
	. "github.com/viant/authz"
	catalog "github.com/viant/datly-studio/studio/resource_policy/catalog"
	"testing"
	"time"
)

type catalogFixture struct {
	rows      []*catalog.Entry
	documents map[Resource]Document
}

func (f *catalogFixture) Catalog(_ context.Context, limit, offset int) ([]*catalog.Entry, error) {
	if offset >= len(f.rows) {
		return nil, nil
	}
	end := offset + limit
	if end > len(f.rows) {
		end = len(f.rows)
	}
	return f.rows[offset:end], nil
}
func (f *catalogFixture) Get(_ context.Context, r Resource) (Document, error) {
	return f.documents[r], nil
}
func (*catalogFixture) Replace(context.Context, Document, int64, string) (Document, error) {
	return Document{}, ErrDenied
}
func TestCatalogFiltersAuthorizationBeforePagination(t *testing.T) {
	fixture := &catalogFixture{documents: map[Resource]Document{}}
	for _, name := range []string{"hidden", "first", "second"} {
		r := Resource{Kind: "component", ID: name, Tenant: "one", Version: "2"}
		subject := "alice"
		if name == "hidden" {
			subject = "bob"
		}
		fixture.rows = append(fixture.rows, &catalog.Entry{Kind: r.Kind, ID: name, Name: name, Tenant: r.Tenant, Version: r.Version, PolicyKind: r.Kind, PolicyID: r.ID})
		fixture.documents[r] = Document{Resource: r, Revision: 1, Policies: map[string]Policy{"viewAccess": {Mode: "protected", Rule: &Rule{Kind: "subject", Value: subject}}}}
	}
	service := &Catalog{Service: &Service{Store: fixture, Provider: catalogTestProvider{Facts{Subject: "alice", Tenant: "one", Issuer: "issuer", ValidUntil: time.Now().Add(time.Minute)}}}}
	page, err := service.List(context.Background(), CatalogInput{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "first" || !page.HasMore {
		t.Fatalf("unauthorized row affected page: %+v", page)
	}
	page, err = service.List(context.Background(), CatalogInput{Limit: 1, Offset: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "second" || page.HasMore {
		t.Fatalf("pagination mismatch: %+v", page)
	}
}

func TestCatalogPublicClassification(t *testing.T) {
	for _, item := range []struct {
		name string
		row  catalog.Entry
		want bool
	}{
		{"no ACL", catalog.Entry{}, true},
		{"scope binding", catalog.Entry{SourceDQL: "component/GET:/_studio/access/context/example/project"}, false},
		{"role", catalog.Entry{HasPolicy: true, PolicyJSON: `{"execute":{"mode":"protected","rule":{"kind":"role","value":"analyst"}}}`}, false},
		{"public consumption, managed changes", catalog.Entry{HasPolicy: true, PolicyJSON: `{"execute":{"mode":"public"},"manageAccess":{"mode":"protected","rule":{"kind":"role","value":"admin"}}}`}, true},
		{"invalid policy", catalog.Entry{HasPolicy: true, PolicyJSON: `invalid`}, false},
		{"no consumption grant", catalog.Entry{HasPolicy: true, PolicyJSON: `{}`}, false},
	} {
		t.Run(item.name, func(t *testing.T) {
			if got := CatalogPublic(&item.row); got != item.want {
				t.Fatalf("public=%v want %v", got, item.want)
			}
		})
	}
}

type catalogTestProvider struct{ facts Facts }

func (p catalogTestProvider) Resolve(context.Context) (Facts, error) { return p.facts, nil }
