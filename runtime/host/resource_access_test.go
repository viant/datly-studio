package host

import (
	"context"
	"testing"
	"time"

	"github.com/viant/datly-studio/sdk/access"
)

type resourceStore struct {
	documents map[access.Resource]access.Document
}

func (s resourceStore) Get(_ context.Context, r access.Resource) (access.Document, error) {
	d, ok := s.documents[r]
	if !ok {
		return d, access.ErrDenied
	}
	return d, nil
}
func (resourceStore) Replace(context.Context, access.Document, int64, string) (access.Document, error) {
	return access.Document{}, access.ErrDenied
}

type runtimeFacts struct{}

func (runtimeFacts) Resolve(context.Context) (access.Facts, error) {
	return access.Facts{Subject: "alice", Tenant: "one", Issuer: "issuer", Roles: []string{"reader"}, Entities: []access.Entity{{Type: "project", ID: "101"}}, ValidUntil: time.Now().Add(time.Minute)}, nil
}

func TestResourceGuardsAndTypedScope(t *testing.T) {
	public := access.Resource{Kind: "skill", ID: "guide", Version: "1", Tenant: "one"}
	private := access.Resource{Kind: "skill", ID: "private", Version: "1", Tenant: "one"}
	role := &access.Rule{Kind: "role", Value: "reader"}
	docs := map[access.Resource]access.Document{
		public:  {Resource: public, Revision: 1, Policies: map[string]access.Policy{"retrieve": {Mode: "protected", Rule: role}}},
		private: {Resource: private, Revision: 1, Policies: map[string]access.Policy{"execute": {Mode: "protected", Rule: role}}},
	}
	s := &Service{config: Config{Access: &ResourceAccessConfig{ResourceBindings: map[string]access.Resource{"skill://guide/": public, "skill://guide/private/": private}}}, resourceAccess: &access.Service{Store: resourceStore{documents: docs}, Provider: runtimeFacts{}}}
	if err := s.authorizeBoundResource(context.Background(), "skill://guide/readme.md"); err != nil {
		t.Fatal(err)
	}
	for _, uri := range []string{"skill://guide/private/readme.md", "skill://guide/../private/readme.md", "skill://guide/%2e%2e/private/readme.md", "skill://unknown/readme.md"} {
		if err := s.authorizeBoundResource(context.Background(), uri); err == nil {
			t.Fatalf("unauthorized URI %s", uri)
		}
	}
	docs[public].Policies["retrieve"] = access.Policy{Mode: "protected", Rule: role, EntityType: "project"}
	if err := s.authorizeBoundResource(context.Background(), "skill://guide/readme.md"); err == nil {
		t.Fatal("bounded scope discarded")
	}
}
