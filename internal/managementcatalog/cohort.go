package managementcatalog

import (
	"encoding/json"
	"fmt"

	ds "github.com/viant/forge/backend/datasource/contract"
	identity "github.com/viant/forge/backend/resource"
	"github.com/viant/forge/backend/types"
)

// CompiledWindowBundles validates the server-owned management catalog as one
// producer publication candidate and closes each window's declared datasource
// references into its immutable schema-2 payload. It does not read business
// rows or make any per-user authorization decision.
func CompiledWindowBundles(config Config) ([]Definition, error) {
	definitions, err := NamespaceWindow(config)
	if err != nil {
		return nil, err
	}
	return CompileDefinitions(definitions)
}

// CompileDefinitions uses the final active management definition set, including
// operator-registered exact-URI overrides, as its sole publication candidate.
func CompileDefinitions(definitions []Definition) ([]Definition, error) {
	byURI := make(map[string]Definition, len(definitions))
	for _, definition := range definitions {
		byURI[definition.URI] = definition
	}
	result := make([]Definition, 0)
	for _, definition := range definitions {
		uri, err := identity.ParseResourceURI(definition.URI)
		if err != nil {
			return nil, err
		}
		if uri.Kind != "window" {
			continue
		}
		closed, err := CompileWindowBundle(definition, byURI)
		if err != nil {
			return nil, err
		}
		result = append(result, Definition{URI: definition.URI, JSON: closed})
	}
	return result, nil
}

// CompileWindowBundle checks every declared child against the same producer
// snapshot, then removes those transport references. Missing children or drifted
// datasource descriptors reject the candidate before it replaces a published root.
func CompileWindowBundle(root Definition, catalog map[string]Definition) (json.RawMessage, error) {
	uri, err := identity.ParseResourceURI(root.URI)
	if err != nil || uri.Kind != "window" {
		return nil, fmt.Errorf("management window root is invalid")
	}
	var envelope types.WindowResourceEnvelope
	if err = json.Unmarshal(root.JSON, &envelope); err != nil || envelope.SchemaVersion != 3 || envelope.Format != types.WindowReferencesFormat || envelope.Validate() != nil {
		return nil, fmt.Errorf("management window source is invalid")
	}
	variants := make(map[string]types.WindowResourceVariant, len(envelope.Variants))
	rebound := make(map[string]string, len(envelope.Variants))
	for oldKey, variant := range envelope.Variants {
		if len(variant.DataSourceResources) != len(variant.DataSources) {
			return nil, fmt.Errorf("management window has unclosed datasource references")
		}
		for id, reference := range variant.DataSourceResources {
			child, ok := catalog[reference.Resource.URI]
			if !ok {
				return nil, fmt.Errorf("management window datasource is missing or changed")
			}
			childURI, parseErr := identity.ParseResourceURI(child.URI)
			if parseErr != nil || childURI.Kind != "datasource" || childURI.Namespace != uri.Namespace {
				return nil, fmt.Errorf("management window datasource identity is invalid")
			}
			var published struct {
				SchemaVersion int             `json:"schemaVersion"`
				DataSource    json.RawMessage `json:"dataSource"`
			}
			if json.Unmarshal(child.JSON, &published) != nil || published.SchemaVersion != 1 {
				return nil, fmt.Errorf("management window datasource payload is invalid")
			}
			inline, inlineErr := types.WindowDescriptorFingerprint(variant.DataSources[id])
			stored, storedErr := types.WindowDescriptorFingerprint(published.DataSource)
			if inlineErr != nil || storedErr != nil || inline != stored {
				return nil, fmt.Errorf("management window inline datasource drifted")
			}
			var executable ds.DataSource
			if json.Unmarshal(variant.DataSources[id], &executable) != nil || executable.Backend == nil {
				return nil, fmt.Errorf("management window datasource backend is invalid")
			}
			switch executable.Backend.Kind {
			case ds.BackendMCPTool:
				// Route the managing SDK operation through this same trusted
				// provider connection in the published host.
			case ds.BackendKind("mcp_protocol"):
				// The native UI selects a runtime MCP endpoint out of band.
				// A complete host artifact instead uses the existing SDK tool,
				// which resolves a requested namespace under the verified actor.
				if executable.Backend.Service != "runtime" || executable.Backend.Method != "tools/list" {
					return nil, fmt.Errorf("unsupported management protocol datasource")
				}
				executable.Backend.Kind = ds.BackendMCPTool
				executable.Backend.Method = "studio.sdk.catalog.tools.list"
			default:
				continue
			}
			if reference.ProviderIdentity == "" {
				return nil, fmt.Errorf("management window MCP provider identity is missing")
			}
			// The complete published artifact is bound to its trusted host MCP
			// connection. The original native management definition remains
			// unchanged; only this producer-approved executable closure uses the
			// provider identity that Agently registered for the same MCP source.
			executable.Backend.Service = reference.ProviderIdentity
			encoded, err := json.Marshal(executable)
			if err != nil {
				return nil, err
			}
			encoded, err = types.CanonicalWindowDescriptor(encoded)
			if err != nil {
				return nil, err
			}
			variant.DataSources[id] = encoded
			fingerprint, err := types.WindowDescriptorFingerprint(encoded)
			if err != nil {
				return nil, err
			}
			variant.Window.ResourceDependencies[id] = fingerprint
		}
		variant.DataSourceResources = nil
		fingerprint, err := types.WindowVariantFingerprint(variant)
		if err != nil {
			return nil, err
		}
		variants[fingerprint] = variant
		rebound[oldKey] = fingerprint
	}
	for i := range envelope.Targets {
		envelope.Targets[i].Variant = rebound[envelope.Targets[i].Variant]
	}
	envelope.SchemaVersion = 2
	envelope.Format = types.WindowBundleFormat
	envelope.Variants = variants
	if err := envelope.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(envelope)
}
