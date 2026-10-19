// Package managementcatalog builds portable administrative resource definitions.
// Building definitions does not authorize, register or expose them; the managing
// MCP must apply its access policy before indexing or returning these bytes.
package managementcatalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	ds "github.com/viant/forge/backend/datasource/contract"
	"github.com/viant/forge/backend/provider/contract"
	identity "github.com/viant/forge/backend/resource"
	"github.com/viant/forge/backend/types"
	"go.yaml.in/yaml/v3"
)

// Config contains host-owned identities, never values supplied by tool callers.
type Config struct {
	Namespace        string
	ProviderIdentity string
	MCPService       string
	PublicURL        string
	ApplicationID    string
	ApplicationName  string
	ApplicationIcon  string
	Sections         []Section
}

type Definition struct {
	URI  string
	JSON json.RawMessage
}

//go:embed namespaces.yaml
var namespacesYAML []byte

// NamespaceWindow returns the management navigation, namespace window bundle,
// and reusable datasource.
// The configured MCP service is the existing management connection; no recursive
// or self-referential MCP configuration resource is emitted.
func NamespaceWindow(config Config) ([]Definition, error) {
	if strings.TrimSpace(config.ProviderIdentity) != config.ProviderIdentity || config.ProviderIdentity == "" || strings.TrimSpace(config.MCPService) != config.MCPService || config.MCPService == "" {
		return nil, fmt.Errorf("management provider and MCP service identities are required")
	}
	windowURI := "window://" + config.Namespace + "/namespaces"
	datasourceURI := "datasource://" + config.Namespace + "/namespaces"
	for _, uri := range []string{windowURI, datasourceURI} {
		parsed, err := identity.ParseResourceURI(uri)
		if err != nil || parsed.Namespace != config.Namespace || parsed.Name != "namespaces" {
			return nil, fmt.Errorf("invalid management resource namespace")
		}
	}
	var source struct {
		Window     *types.Window  `yaml:"window"`
		DataSource *ds.DataSource `yaml:"dataSource"`
	}
	if err := yaml.Unmarshal(namespacesYAML, &source); err != nil {
		return nil, err
	}
	source.Window.Namespace = config.Namespace
	source.DataSource.Backend.Service = config.MCPService
	descriptor, err := json.Marshal(source.DataSource)
	if err != nil {
		return nil, err
	}
	descriptor, err = types.CanonicalWindowDescriptor(descriptor)
	if err != nil {
		return nil, err
	}
	descriptorFingerprint, err := types.WindowDescriptorFingerprint(descriptor)
	if err != nil {
		return nil, err
	}
	datasource, err := json.Marshal(struct {
		SchemaVersion int             `json:"schemaVersion"`
		DataSource    json.RawMessage `json:"dataSource"`
	}{1, descriptor})
	if err != nil {
		return nil, err
	}
	// MCP clients may decode structured content into object maps. Canonicalize
	// the entire reusable resource before hashing, not only its inner descriptor.
	datasource, err = types.CanonicalWindowDescriptor(datasource)
	if err != nil {
		return nil, err
	}
	id := source.DataSource.ID
	source.Window.DataSource = map[string]types.DataSource{id: source.DataSource.DataSource}
	source.Window.ResourceDependencies = map[string]string{id: descriptorFingerprint}
	variant := types.WindowResourceVariant{
		Window:      source.Window,
		DataSources: map[string]json.RawMessage{id: descriptor},
		DataSourceResources: map[string]primitive.DataSourceReference{id: {
			Resource:           identity.ResourceRef{URI: datasourceURI, Revision: identity.WorkingCandidate},
			ContentFingerprint: identity.ContentFingerprint(datasource),
			ProviderIdentity:   config.ProviderIdentity,
		}},
	}
	fingerprint, err := types.WindowVariantFingerprint(variant)
	if err != nil {
		return nil, err
	}
	envelope := types.WindowResourceEnvelope{
		SchemaVersion: 3, Format: types.WindowReferencesFormat,
		Targets:  []types.WindowTargetBinding{{Variant: fingerprint}, {Target: types.WindowTarget{Platform: "web", FormFactor: "desktop"}, Variant: fingerprint}},
		Variants: map[string]types.WindowResourceVariant{fingerprint: variant},
	}
	if err := envelope.Validate(); err != nil {
		return nil, err
	}
	window, err := json.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	window, err = types.CanonicalWindowDescriptor(window)
	if err != nil {
		return nil, err
	}
	nav, err := navigation(config)
	if err != nil {
		return nil, err
	}
	return []Definition{{URI: windowURI, JSON: window}, {URI: datasourceURI, JSON: datasource}, nav}, nil
}
