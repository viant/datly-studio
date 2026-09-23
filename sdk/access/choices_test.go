package access

import (
	"context"
	"testing"
	"time"
)

func TestEditorContextRequiresViewAndSeparatesManagement(t *testing.T) {
	r := Resource{Kind: "skill", ID: "one", Tenant: "one", Version: "1"}
	p := Policy{Mode: "protected", Rule: &Rule{Kind: "role", Value: "reader"}}
	s := Service{Store: &testStore{doc: Document{Resource: r, Revision: 1, Policies: map[string]Policy{"viewAccess": p}}}, Provider: testProvider{Facts{Subject: "alice", Tenant: "one", Issuer: "issuer", Roles: []string{"reader"}, Exposures: []string{"analytics"}, Entities: []Entity{{"project", "101"}}, ValidUntil: time.Now().Add(time.Minute)}}}
	result, err := s.EditorContext(context.Background(), r)
	if err != nil || result.CanManage || len(result.Choices.Roles) != 1 || result.Choices.Entities[0].Entity.Type != "project" {
		t.Fatalf("unexpected context %+v %v", result, err)
	}
	r.Tenant = "two"
	if _, err = s.EditorContext(context.Background(), r); err == nil {
		t.Fatal("cross tenant directory disclosure")
	}
}
