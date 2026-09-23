package access

import (
	"context"
	"testing"
	"time"
)

type testProvider struct{ facts Facts }

func (p testProvider) Resolve(context.Context) (Facts, error) { return p.facts, nil }

type testStore struct {
	doc      Document
	writes   int
	conflict bool
}

func (s *testStore) Get(context.Context, Resource) (Document, error) { return s.doc, nil }
func (s *testStore) Replace(_ context.Context, d Document, revision int64, subject string) (Document, error) {
	if s.conflict || revision != s.doc.Revision {
		return Document{}, ErrConflict
	}
	s.writes++
	d.Revision++
	s.doc = d
	return d, nil
}

func TestPolicyAdministration(t *testing.T) {
	r := Resource{Kind: "skill", ID: "inspect", Version: "1", Tenant: "one"}
	admin := Policy{Mode: "protected", Rule: &Rule{Kind: "subject", Value: "alice"}}
	store := &testStore{doc: Document{Resource: r, Revision: 1, Policies: map[string]Policy{"viewAccess": admin, "manageAccess": admin}}}
	s := Service{Store: store, Provider: testProvider{Facts{Subject: "bob", Tenant: "one", Issuer: "issuer", ValidUntil: time.Now().Add(time.Minute)}}}
	candidate := Document{Resource: r, Revision: 1, Policies: map[string]Policy{"retrieve": {Mode: "public"}, "manageAccess": admin, "viewAccess": admin}}
	if _, err := s.Replace(context.Background(), candidate); err == nil || store.writes != 0 {
		t.Fatal("unauthorized mutation")
	}
	s.Provider = testProvider{Facts{Subject: "alice", Tenant: "one", Issuer: "issuer", ValidUntil: time.Now().Add(time.Minute)}}
	store.conflict = true
	if _, err := s.Replace(context.Background(), candidate); err != ErrConflict || store.writes != 0 {
		t.Fatal("concurrent authorization change lost")
	}
	store.conflict = false
	if doc, err := s.Replace(context.Background(), candidate); err != nil || doc.Revision != 2 || store.writes != 1 {
		t.Fatalf("replace failed: %+v %v", doc, err)
	}
	if _, err := s.Replace(context.Background(), candidate); err != ErrConflict || store.writes != 1 {
		t.Fatal("stale revision accepted")
	}
}
