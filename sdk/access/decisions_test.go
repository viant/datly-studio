package access

import (
	"context"
	"testing"
	"time"
)

type decisionFunc func(context.Context, Request, Document, Facts) (Decision, error)

func (f decisionFunc) Evaluate(c context.Context, r Request, d Document, p Facts) (Decision, error) {
	return f(c, r, d, p)
}

func TestRemoteDecisionBoundary(t *testing.T) {
	r := Resource{Kind: "component", ID: "data", Tenant: "one", Version: "1"}
	policy := Policy{Mode: "protected", Rule: &Rule{Kind: "role", Value: "reader"}}
	facts := Facts{Subject: "alice", Tenant: "one", Issuer: "issuer", Roles: []string{"reader"}, ValidUntil: time.Now().Add(time.Minute)}
	store := &testStore{doc: Document{Resource: r, Revision: 1, Policies: map[string]Policy{"viewAccess": policy, "manageAccess": policy}}}
	s := Service{Store: store, Provider: testProvider{facts: facts}, Decisions: decisionFunc(func(_ context.Context, r Request, _ Document, _ Facts) (Decision, error) {
		if r.Action == "manageAccess" {
			return Decision{}, ErrDenied
		}
		return Decision{}, nil
	})}
	editor, err := s.EditorContext(context.Background(), r)
	if err != nil || editor.CanManage {
		t.Fatalf("UI ignored provider denial: %+v %v", editor, err)
	}
	if _, err = s.Replace(context.Background(), store.doc); err == nil || store.writes != 0 {
		t.Fatal("provider denial permitted mutation")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	s.Decisions = decisionFunc(func(context.Context, Request, Document, Facts) (Decision, error) { cancel(); return Decision{}, nil })
	if _, err = s.Authorize(cancelled, Request{Resource: r, Action: "viewAccess"}); err == nil {
		t.Fatal("cancelled provider call allowed")
	}
}

func TestPublicConsumptionDoesNotDependOnRemoteProvider(t *testing.T) {
	r := Resource{Kind: "component", ID: "public", Tenant: "*", Version: "1"}
	s := Service{Store: &testStore{doc: Document{Resource: r, Revision: 1, Policies: map[string]Policy{"execute": {Mode: "public"}}}}, Decisions: decisionFunc(func(context.Context, Request, Document, Facts) (Decision, error) {
		t.Fatal("public policy called remote provider")
		return Decision{}, ErrDenied
	})}
	if _, err := s.Authorize(context.Background(), Request{Resource: r, Action: "execute"}); err != nil {
		t.Fatal(err)
	}
}
