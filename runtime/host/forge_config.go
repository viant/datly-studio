package host

import (
	"fmt"
	"path/filepath"
	"strings"

	identity "github.com/viant/agently-core/protocol/resource"
	"github.com/viant/authz"
)

// ForgeConfig enables the generic stock Forge window provider for a dynamic
// Datly host. Definitions and ACL policy are operator-owned; Studio does not
// add reporting-specific configuration or storage.
type ForgeConfig struct {
	Windows    []ForgeWindow             `json:"windows" yaml:"Windows"`
	Policies   []authz.Document          `json:"policies" yaml:"Policies"`
	Selections []authz.SelectionDocument `json:"selections" yaml:"Selections"`
}

// ForgeWindow maps one logical window to its unversioned YAML or JSON source
// and exact Datly datasource components. Client requests cannot alter it.
type ForgeWindow struct {
	URI            string                        `json:"uri" yaml:"URI"`
	DefinitionPath string                        `json:"definitionPath" yaml:"DefinitionPath"`
	Components     map[string]ComponentReference `json:"components" yaml:"Components"`
}

func validateForgeConfig(config *Config) error {
	if config.Forge == nil {
		return nil
	}
	if config.ForgeProvider != nil {
		return fmt.Errorf("Forge config and an injected Forge provider cannot both be configured")
	}
	if config.Access == nil || config.Access.Tenant == "" {
		return fmt.Errorf("stock Forge windows require resource access with a tenant")
	}
	if len(config.Forge.Windows) == 0 || len(config.Forge.Windows) > 100 {
		return fmt.Errorf("stock Forge config requires between one and 100 windows")
	}
	if len(config.Forge.Policies) == 0 || len(config.Forge.Selections) == 0 {
		return fmt.Errorf("stock Forge windows require explicit policies and selections")
	}
	windows := make(map[string]ForgeWindow, len(config.Forge.Windows))
	families := make(map[authz.ResourceFamily]bool, len(config.Forge.Windows))
	for _, window := range config.Forge.Windows {
		uri, err := identity.ParseResourceURI(window.URI)
		if err != nil || uri.Kind != "window" {
			return fmt.Errorf("stock Forge resource URI must identify a window")
		}
		if _, exists := windows[window.URI]; exists {
			return fmt.Errorf("duplicate stock Forge window URI %q", window.URI)
		}
		if !validForgeDefinitionPath(window.DefinitionPath) {
			return fmt.Errorf("stock Forge window %s requires a relative DefinitionPath", window.URI)
		}
		if len(window.Components) == 0 {
			return fmt.Errorf("stock Forge window %s requires explicit datasource components", window.URI)
		}
		for datasourceID, ref := range window.Components {
			if !portableSourceID(datasourceID) {
				return fmt.Errorf("invalid stock Forge datasource id %q", datasourceID)
			}
			if err := validateComponentReference(ref); err != nil {
				return fmt.Errorf("stock Forge datasource %s: %w", datasourceID, err)
			}
		}
		family := authz.ResourceFamily{Kind: "window", ID: uri.Namespace + "/" + uri.Name, Tenant: config.Access.Tenant}
		families[family] = true
		windows[window.URI] = window
	}
	if _, err := authz.NewStaticStore(config.Forge.Policies); err != nil {
		return fmt.Errorf("stock Forge policies: %w", err)
	}
	seenPolicies := map[authz.Resource]bool{}
	for _, document := range config.Forge.Policies {
		resource := document.Resource
		family := authz.ResourceFamily{Kind: resource.Kind, ID: resource.ID, Tenant: resource.Tenant}
		if !families[family] || resource.Version != identity.WorkingCandidate || seenPolicies[resource] {
			return fmt.Errorf("stock Forge policy is not bound to a configured working window")
		}
		seenPolicies[resource] = true
		for _, action := range []string{"discover", "describe", "execute"} {
			if _, ok := document.Policies[action]; !ok {
				return fmt.Errorf("stock Forge policy %s requires %s", resource.ID, action)
			}
		}
	}
	if len(seenPolicies) != len(windows) {
		return fmt.Errorf("each stock Forge window requires one exact working policy")
	}
	if _, err := authz.NewStaticSelectionStore(config.Forge.Selections); err != nil {
		return fmt.Errorf("stock Forge selections: %w", err)
	}
	seenFamilies := map[authz.ResourceFamily]bool{}
	for _, selection := range config.Forge.Selections {
		if !families[selection.Resource] || selection.DefaultVersion != identity.WorkingCandidate || len(selection.Overrides) != 0 || seenFamilies[selection.Resource] {
			return fmt.Errorf("stock Forge selections must explicitly select each working window")
		}
		seenFamilies[selection.Resource] = true
	}
	if len(seenFamilies) != len(windows) {
		return fmt.Errorf("each stock Forge window requires one explicit working selection")
	}
	return nil
}

func validForgeDefinitionPath(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && !filepath.IsAbs(value) && filepath.Clean(value) == value && value != "." && value != ".." && !strings.HasPrefix(value, ".."+string(filepath.Separator))
}

func portableSourceID(value string) bool {
	if value == "" || strings.TrimSpace(value) != value {
		return false
	}
	for index, r := range value {
		if index == 0 && !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}
