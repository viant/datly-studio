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

func TestAuthorizeWithFactsReturnsDecisionSnapshot(t *testing.T) {
	resource := Resource{Kind: "dataSource", ID: "operations", Version: "1", Tenant: "one"}
	policy := Policy{Mode: "protected", Rule: &Rule{Kind: "role", Value: "analyst"}, EntityType: "project"}
	facts := Facts{Subject: "alice", Tenant: "one", Issuer: "issuer", Roles: []string{"analyst"}, Exposures: []string{"reports"}, EntityGroups: EntityGroups{"project": {"101", "102"}}, ValidUntil: time.Now().Add(time.Hour)}
	service := &Service{Store: &testStore{doc: Document{Resource: resource, Revision: 1, Policies: map[string]Policy{"execute": policy}}}, Provider: testProvider{facts}}
	decision, used, err := service.AuthorizeWithFacts(context.Background(), Request{Resource: resource, Action: "execute"})
	if err != nil || !decision.Bounded || len(decision.Entities) != 2 || used.Subject != "alice" || len(used.Roles) != 1 || used.Roles[0] != "analyst" {
		t.Fatalf("decision and verified facts diverged: %+v %+v %v", decision, used, err)
	}
	service.Provider = testProvider{Facts{Subject: "bob", Tenant: "one", Issuer: "issuer", Roles: []string{"guest"}, ValidUntil: time.Now().Add(time.Hour)}}
	decision, used, err = service.AuthorizeWithFacts(context.Background(), Request{Resource: resource, Action: "execute"})
	if err == nil || decision.Bounded || used.Subject != "" {
		t.Fatalf("denied request exposed facts: %+v %+v %v", decision, used, err)
	}
}

func TestReportPreviewAndExecutionStayIndependentAcrossRoleExposureAndEntities(t *testing.T) {
	resource := Resource{Kind: "report", ID: "operations", Version: "3", Tenant: "one"}
	preview := Policy{Mode: "protected", EntityType: "project", Rule: &Rule{Kind: "all", Rules: []Rule{
		{Kind: "role", Value: "reviewer"}, {Kind: "exposure", Value: "analytics"},
	}}}
	execute := Policy{Mode: "protected", Rule: &Rule{Kind: "role", Value: "runner"}}
	facts := Facts{Subject: "alice", Tenant: "one", Issuer: "issuer", Roles: []string{"reviewer"}, Exposures: []string{"analytics"},
		EntityGroups: EntityGroups{"project": {"101", "102"}}, ValidUntil: time.Now().Add(time.Hour)}
	service := &Service{Store: &testStore{doc: Document{Resource: resource, Revision: 1, Policies: map[string]Policy{"preview": preview, "execute": execute}}}, Provider: testProvider{facts}}
	decision, err := service.Authorize(context.Background(), Request{Resource: resource, Action: "preview"})
	if err != nil || !decision.Bounded || len(decision.Entities) != 2 {
		t.Fatalf("report preview scope=%+v err=%v", decision, err)
	}
	if _, err := service.Authorize(context.Background(), Request{Resource: resource, Action: "execute"}); err == nil {
		t.Fatal("preview permission incorrectly granted execution")
	}
	facts.Roles = []string{"runner"}
	service.Provider = testProvider{facts}
	if _, err := service.Authorize(context.Background(), Request{Resource: resource, Action: "preview"}); err == nil {
		t.Fatal("execution role incorrectly granted preview")
	}
	decision, err = service.Authorize(context.Background(), Request{Resource: resource, Action: "execute"})
	if err != nil || decision.Bounded {
		t.Fatalf("runner execution=%+v err=%v", decision, err)
	}
}
