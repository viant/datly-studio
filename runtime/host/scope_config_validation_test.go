package host

import "testing"

func TestScopeBindingValidateRequiresCanonicalPositiveVersion(t *testing.T) {
	for _, version := range []string{"0", "-1", "01", "+1"} {
		t.Run("reject_"+version, func(t *testing.T) {
			binding := ScopeBinding{Component: "tasks", Version: version, EntityType: "project", Parameter: "Scope"}
			if err := binding.validate(); err == nil {
				t.Fatalf("accepted version %q", version)
			}
		})
	}
	for _, version := range []string{"1", "123"} {
		t.Run("accept_"+version, func(t *testing.T) {
			binding := ScopeBinding{Component: "tasks", Version: version, EntityType: "project", Parameter: "Scope"}
			if err := binding.validate(); err != nil {
				t.Fatalf("rejected version %q: %v", version, err)
			}
		})
	}
}

func TestScopeBindingValidateRequiresGoStyleParameterIdentifier(t *testing.T) {
	for _, parameter := range []string{"1Scope"} {
		t.Run("reject_"+parameter, func(t *testing.T) {
			binding := ScopeBinding{Component: "tasks", Version: "1", EntityType: "project", Parameter: parameter}
			if err := binding.validate(); err == nil {
				t.Fatalf("accepted parameter %q", parameter)
			}
		})
	}
	for _, parameter := range []string{"Scope_1", "_Scope"} {
		t.Run("accept_"+parameter, func(t *testing.T) {
			binding := ScopeBinding{Component: "tasks", Version: "1", EntityType: "project", Parameter: parameter}
			if err := binding.validate(); err != nil {
				t.Fatalf("rejected parameter %q: %v", parameter, err)
			}
		})
	}
}
