package settings

import (
	"encoding/json"
	"testing"
)

func TestDecodeInternalComponentRestriction(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		want         bool
	}{
		{"existing settings", `{"predicatePackages":[],"requireSecurityPredicate":false}`, false},
		{"enabled", `{"requireInternalComponents":true}`, true},
		{"disabled", `{"requireInternalComponents":false}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, err := Decode(SecurityKey, json.RawMessage(tc.source))
			if err != nil {
				t.Fatal(err)
			}
			if got := value.(*Security).RequireInternalComponents; got != tc.want {
				t.Fatalf("restriction = %v, want %v", got, tc.want)
			}
		})
	}
	if _, err := Decode(SecurityKey, json.RawMessage(`{"requireInternalComponents":"true"}`)); err == nil {
		t.Fatal("string value must not be accepted as a boolean policy")
	}
}
