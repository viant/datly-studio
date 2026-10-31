// Package managementwindow builds portable Forge window and datasource resources.
// It does not authorize or publish them; the managing producer owns admission.
package managementwindow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	ds "github.com/viant/forge/backend/datasource/contract"
	"github.com/viant/forge/backend/provider/contract"
	identity "github.com/viant/forge/backend/resource"
	"github.com/viant/forge/backend/types"
)

type Config struct{ Namespace, ProviderIdentity, MCPService string }
type Definition struct {
	URI  string
	JSON json.RawMessage
}

// Build binds every datasource to one existing producer connection and returns
// a format-3 window plus reusable logical datasource resources.
func Build(config Config, name string, window *types.Window, sources map[string]*ds.DataSource) ([]Definition, error) {
	if window == nil || config.ProviderIdentity == "" || strings.TrimSpace(config.ProviderIdentity) != config.ProviderIdentity || config.MCPService == "" || strings.TrimSpace(config.MCPService) != config.MCPService {
		return nil, fmt.Errorf("management window, provider and MCP service required")
	}
	windowURI := "window://" + config.Namespace + "/" + name
	parsed, err := identity.ParseResourceURI(windowURI)
	if err != nil || parsed.Namespace != config.Namespace || parsed.Name != name {
		return nil, fmt.Errorf("invalid management window identity")
	}
	window.Namespace = config.Namespace
	window.DataSource = map[string]types.DataSource{}
	window.ResourceDependencies = map[string]string{}
	variant := types.WindowResourceVariant{Window: window, DataSources: map[string]json.RawMessage{}, DataSourceResources: map[string]primitive.DataSourceReference{}}
	ids := make([]string, 0, len(sources))
	for id := range sources {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var definitions []Definition
	for _, id := range ids {
		source := sources[id]
		if source == nil || source.ID != id || source.Backend == nil {
			return nil, fmt.Errorf("invalid management datasource %s", id)
		}
		uri := "datasource://" + config.Namespace + "/" + id
		parsed, err := identity.ParseResourceURI(uri)
		if err != nil || parsed.Namespace != config.Namespace || parsed.Name != id {
			return nil, fmt.Errorf("invalid management datasource identity")
		}
		if source.Backend.Kind == ds.BackendMCPTool {
			source.Backend.Service = config.MCPService
		}
		descriptor, err := json.Marshal(source)
		if err != nil {
			return nil, err
		}
		descriptor, err = types.CanonicalWindowDescriptor(descriptor)
		if err != nil {
			return nil, err
		}
		fingerprint, err := types.WindowDescriptorFingerprint(descriptor)
		if err != nil {
			return nil, err
		}
		resource, err := json.Marshal(struct {
			SchemaVersion int             `json:"schemaVersion"`
			DataSource    json.RawMessage `json:"dataSource"`
		}{1, descriptor})
		if err != nil {
			return nil, err
		}
		resource, err = types.CanonicalWindowDescriptor(resource)
		if err != nil {
			return nil, err
		}
		window.DataSource[id] = source.DataSource
		window.ResourceDependencies[id] = fingerprint
		variant.DataSources[id] = descriptor
		variant.DataSourceResources[id] = primitive.DataSourceReference{Resource: identity.ResourceRef{URI: uri, Revision: identity.WorkingCandidate}, ProviderIdentity: config.ProviderIdentity}
		definitions = append(definitions, Definition{URI: uri, JSON: resource})
	}
	fingerprint, err := types.WindowVariantFingerprint(variant)
	if err != nil {
		return nil, err
	}
	envelope := types.WindowResourceEnvelope{SchemaVersion: 3, Format: types.WindowReferencesFormat, Targets: []types.WindowTargetBinding{{Variant: fingerprint}, {Target: types.WindowTarget{Platform: "web", FormFactor: "desktop"}, Variant: fingerprint}}, Variants: map[string]types.WindowResourceVariant{fingerprint: variant}}
	if err := envelope.Validate(); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	encoded, err = types.CanonicalWindowDescriptor(encoded)
	if err != nil {
		return nil, err
	}
	return append([]Definition{{URI: windowURI, JSON: encoded}}, definitions...), nil
}
