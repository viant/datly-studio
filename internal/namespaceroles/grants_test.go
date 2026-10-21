package namespaceroles

import "testing"

func TestLegacyRolesRemainViewOnly(t *testing.T) {
	raw := `["reader"]`
	g, err := Decode(&raw)
	if err != nil || len(g.View) != 1 || len(g.Manage) != 0 {
		t.Fatalf("legacy grants: %+v %v", g, err)
	}
	encoded, err := g.Encode()
	if err != nil || encoded != raw {
		t.Fatalf("legacy encoding: %s %v", encoded, err)
	}
	g.Manage = []string{"TEAM_MANAGER"}
	encoded, err = g.Encode()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Decode(&encoded)
	if err != nil || !Matches(restored.Manage, []string{"TEAM_MANAGER"}) || Matches(restored.Manage, []string{"team_manager"}) {
		t.Fatalf("manager round trip: %+v %v", restored, err)
	}
}
func TestInvalidGrantsFailClosed(t *testing.T) {
	for _, raw := range []string{`null`, `{"manage":["*"]}`, `{"manage":["admin","admin"]}`, `{"manage":[" admin"]}`, `{"manage":"admin"}`, `{"unexpected":["admin"]}`, `[] {}`} {
		if _, err := Decode(&raw); err == nil {
			t.Errorf("accepted invalid grants %s", raw)
		}
	}
}
