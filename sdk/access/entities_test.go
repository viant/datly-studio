package access

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestEntityMapPreservesExactIntegerIDs(t *testing.T) {
	groups, err := DecodeEntityGroups([]byte(`{"project":["alpha",9007199254740993,18446744073709551615],"organization":["north"]}`))
	if err != nil {
		t.Fatal(err)
	}
	flat, err := NormalizeEntityGroups(groups)
	if err != nil {
		t.Fatal(err)
	}
	want := []Entity{{Type: "organization", ID: "north"}, {Type: "project", ID: "alpha"}, {Type: "project", ID: "9007199254740993"}, {Type: "project", ID: "18446744073709551615"}}
	if !reflect.DeepEqual(flat, want) {
		t.Fatalf("exact IDs: %+v", flat)
	}
	facts := Facts{EntityGroups: groups, Entities: flat}
	ids, err := facts.IDsForType("project")
	if err != nil || !reflect.DeepEqual(ids, groups["project"]) {
		t.Fatalf("typed IDs %v: %v", ids, err)
	}
	for _, entityType := range []string{"other", "empty"} {
		if ids, err = facts.IDsForType(entityType); err != nil || len(ids) != 0 {
			t.Fatalf("%s: %v %v", entityType, ids, err)
		}
	}
	encoded, err := json.Marshal(facts)
	if err != nil || !bytes.Contains(encoded, []byte(`"allowedEntities":{"organization":["north"],"project":["alpha","9007199254740993","18446744073709551615"]}`)) {
		t.Fatalf("public facts: %s %v", encoded, err)
	}
}

func TestEntityMapRejectsMalformedOrConflictingAuthority(t *testing.T) {
	for _, raw := range []string{
		`null`, `[]`, `{"": [1]}`, `{" project ":[1]}`, `{"pro ject":[1]}`, `{"project/child":[1]}`, `{"project":null}`,
		`{"project":[0]}`, `{"project":[-1]}`, `{"project":[1.5]}`, `{"project":[1e3]}`,
		`{"project":[18446744073709551616]}`, `{"project":[1,"1"]}`,
		`{"project":[1],"project":[2]}`, `{"project":{"ids":[1]}}`, `{"project":[" "]}`,
	} {
		t.Run(raw, func(t *testing.T) {
			if _, err := DecodeEntityGroups([]byte(raw)); !errors.Is(err, ErrDenied) {
				t.Fatalf("accepted malformed map: %v", err)
			}
		})
	}
	facts := Facts{EntityGroups: EntityGroups{"project": {"102"}}, Entities: []Entity{{Type: "project", ID: "101"}}}
	if _, err := facts.FlatEntities(); !errors.Is(err, ErrDenied) {
		t.Fatalf("conflicting views: %v", err)
	}
	if _, err := GroupEntities([]Entity{{Type: "project", ID: "101"}, {Type: "project", ID: "101"}}); !errors.Is(err, ErrDenied) {
		t.Fatalf("duplicate legacy facts: %v", err)
	}
}

func TestEntityMapDrivesPredicateAndScopeWithoutWidening(t *testing.T) {
	now := time.Now()
	r := Resource{Kind: "component", ID: "tasks", Version: "1", Tenant: "one"}
	facts := Facts{Subject: "alice", Tenant: "one", Issuer: "issuer", ValidUntil: now.Add(time.Minute), EntityGroups: EntityGroups{"project": {"101", "102"}}}
	policy := Policy{Mode: "protected", Rule: &Rule{Kind: "entity", Entity: &Entity{Type: "project", ID: "101"}}, EntityType: "project"}
	decision, err := Evaluate(Request{Resource: r, Action: "execute"}, map[string]Policy{"execute": policy}, facts, now)
	if err != nil || !decision.Bounded || len(decision.Entities) != 2 {
		t.Fatalf("map predicate/scope: %+v %v", decision, err)
	}
	facts.EntityGroups = EntityGroups{"project": {}}
	if ids, err := facts.IDsForType("project"); err != nil || len(ids) != 0 {
		t.Fatalf("empty key: %v %v", ids, err)
	}
	if _, err = Evaluate(Request{Resource: r, Action: "execute"}, map[string]Policy{"execute": policy}, facts, now); !errors.Is(err, ErrDenied) {
		t.Fatalf("empty group granted access: %v", err)
	}
	facts.EntityGroups = EntityGroups{"project": {"101"}}
	facts.Entities = []Entity{{Type: "project", ID: "999"}}
	if _, err = Evaluate(Request{Resource: r, Action: "execute"}, map[string]Policy{"execute": policy}, facts, now); !errors.Is(err, ErrDenied) {
		t.Fatalf("conflicting flat fact widened access: %v", err)
	}
}
