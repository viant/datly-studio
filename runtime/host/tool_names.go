package host

import (
	"fmt"
	"strings"

	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
)

// MCPToolNames changes only the published tool's owner prefix. Verified subject,
// component identity, package names, route targets and access policy stay intact.
type MCPToolNames struct {
	RemoveSubjectPhrases []string `yaml:"RemoveSubjectPhrases"`
}

func (c MCPToolNames) toolName(name string) (string, error) {
	dot := strings.IndexByte(name, '.')
	if dot < 1 || len(c.RemoveSubjectPhrases) == 0 {
		return name, nil
	}
	original := name[:dot]
	prefix := strings.ToLower(original)
	for _, phrase := range c.RemoveSubjectPhrases {
		if phrase = strings.ToLower(strings.TrimSpace(phrase)); phrase != "" {
			prefix = strings.ReplaceAll(prefix, phrase, "")
		}
	}
	if prefix == strings.ToLower(original) {
		return name, nil
	}
	if prefix == "" || prefix[0] < 'a' || prefix[0] > 'z' {
		return "", fmt.Errorf("tool-name phrase removal leaves an invalid owner prefix for %q", name)
	}
	return prefix + name[dot:], nil
}

func (c MCPToolNames) registrations(source []*registry.RegisteredComponent) ([]*registry.RegisteredComponent, error) {
	if len(c.RemoveSubjectPhrases) == 0 {
		return source, nil
	}
	result := make([]*registry.RegisteredComponent, len(source))
	names := map[string]bool{}
	for i, registered := range source {
		if registered == nil || registered.Component == nil {
			result[i] = registered
			continue
		}
		current := *registered
		current.Component = registered.Component.Clone()
		result[i] = &current
		for _, route := range current.Component.Routes {
			if route == nil {
				continue
			}
			for _, exposure := range route.MCP {
				if exposure == nil || exposure.Kind != spec.MCPExposureTool {
					continue
				}
				name, err := c.toolName(exposure.Name)
				if err != nil {
					return nil, err
				}
				key := strings.ToLower(name)
				if names[key] {
					return nil, fmt.Errorf("tool-name phrase removal causes a duplicate MCP tool %q", name)
				}
				names[key] = true
				exposure.Name = name
			}
		}
	}
	return result, nil
}
