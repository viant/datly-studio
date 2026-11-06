package host

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/runtime/namespacemcp"

	identity "github.com/viant/forge/backend/resource"
	coreresource "github.com/viant/forge/backend/resourcehost"
	windowprotocol "github.com/viant/forge/backend/window/contract"
)

func (s *stockPrimitiveSource) componentPin(ctx context.Context, ref ComponentReference, compiled map[ComponentReference]stockComponentPin, sources map[ComponentReference]string) (stockComponentPin, error) {
	before, err := s.original.service.primitiveComponentSource(ctx, ref)
	if err != nil {
		return stockComponentPin{}, err
	}
	pin, found := compiled[ref]
	if !found {
		manager, binding, err := s.original.service.componentRuntimeWithAuthority(ctx, ref, false)
		if err != nil {
			return pin, err
		}
		if err = manager.Shutdown(ctx); err != nil {
			return pin, err
		}
		s.metadataCompiles.Add(1)
		if ref.Kind == "dynamic" && binding.ContentFingerprint != before {
			return pin, identity.ErrResourceStale
		}
		pin = stockComponentPin{Reference: ref, Binding: binding}
		compiled[ref] = pin
	}
	after, err := s.original.service.primitiveComponentSource(ctx, ref)
	if err != nil || before != after {
		return pin, identity.ErrResourceStale
	}
	if original, exists := sources[ref]; exists && original != after {
		return pin, identity.ErrResourceStale
	}
	sources[ref] = after
	return pin, nil
}

func (s *stockPrimitiveSource) standaloneDefinition(config ForgeDataSource, compiled map[ComponentReference]stockComponentPin, sources map[ComponentReference]string) coreresource.NativeSnapshotDefinition {
	return coreresource.NativeSnapshotDefinition{URI: config.URI, Title: config.URI, FormatVersion: 1, File: config.DefinitionPath, Load: func(ctx context.Context, reader *coreresource.ExtensionReader, file string) (json.RawMessage, error) {
		raw, err := reader.ReadFile(ctx, file)
		if err != nil {
			return nil, err
		}
		var authored stockDatasourceDefinition
		if json.Unmarshal(raw, &authored) != nil || authored.SchemaVersion != 1 || authored.DataSource == nil || !portableSourceID(authored.DataSource.ID) || authored.DataSource.Backend != nil || authored.DataSource.Service != nil {
			return nil, fmt.Errorf("configured datasource requires an authored descriptor without backend or service")
		}
		pin, err := s.componentPin(ctx, config.Component, compiled, sources)
		if err != nil {
			return nil, err
		}
		authored.DataSource.Backend = &windowprotocol.Backend{Kind: "datly", Ownership: "provider", Method: "datasources/fetch", Component: &pin.Binding, SchemaFingerprint: pin.Binding.SchemaFingerprint}
		raw, err = json.Marshal(authored)
		if err != nil {
			return nil, err
		}
		if err = validateStockDatasourceDefinition(1, raw); err != nil {
			return nil, err
		}
		return raw, nil
	}}
}

func standaloneDatasourceBinding(entry stockDatasourceEntry, canonical *stockPrimitiveSource, policy *stockWindowPolicy, provider string) coreresource.LocalResourceBinding {
	return coreresource.LocalResourceBinding{URI: entry.uri.String(), Title: entry.id, FormatVersion: 1, Changes: canonical.snapshot, Resolver: func(_ context.Context, _ identity.VerifiedActor, action string) (*identity.ResourceResolver, error) {
		selectedAction := "discover"
		if action == "resource.get" {
			selectedAction = "describe"
		}
		return &identity.ResourceResolver{ProviderIdentity: provider, Source: stockDatasourceSource{canonical: canonical, entry: entry}, Policy: stockDatasourcePolicy{base: policy, entry: entry, action: selectedAction}}, nil
	}}
}

// A namespaced runtime serves only resources authored in that namespace. The
// standalone host (no NamespaceID) retains its existing multi-namespace catalog.
func (s *Service) scopeConfiguredForge(ctx context.Context) error {
	if s.config.Forge == nil || s.config.NamespaceID == "" {
		return nil
	}
	var name string
	if s.config.namespaceDefinitions != nil {
		if row, ok := s.config.namespaceDefinitions.Lookup(s.config.NamespaceID); ok {
			name = row.Name
		}
	}
	if name == "" {
		rows, err := (namespacemcp.SQLDefinitions{DB: s.studio}).LoadNamespace(ctx, s.config.NamespaceID)
		if err != nil || len(rows) != 1 {
			return fmt.Errorf("configured Forge namespace is unavailable")
		}
		name = rows[0].Name
	}
	config := *s.config.Forge
	config.Windows = nil
	config.DataSources = nil
	config.Policies = nil
	config.Selections = nil
	selected := map[authz.ResourceFamily]bool{}
	for _, window := range s.config.Forge.Windows {
		uri, _ := identity.ParseResourceURI(window.URI)
		if uri.Namespace == name {
			config.Windows = append(config.Windows, window)
			selected[authz.ResourceFamily{Kind: "window", ID: uri.Namespace + "/" + uri.Name, Tenant: s.config.Access.Tenant}] = true
		}
	}
	for _, source := range s.config.Forge.DataSources {
		uri, _ := identity.ParseResourceURI(source.URI)
		if uri.Namespace == name {
			config.DataSources = append(config.DataSources, source)
			selected[authz.ResourceFamily{Kind: "datasource", ID: uri.Namespace + "/" + uri.Name, Tenant: s.config.Access.Tenant}] = true
		}
	}
	if len(selected) == 0 {
		s.config.Forge = nil
		return nil
	}
	for _, document := range s.config.Forge.Policies {
		r := document.Resource
		if selected[authz.ResourceFamily{Kind: r.Kind, ID: r.ID, Tenant: r.Tenant}] {
			config.Policies = append(config.Policies, document)
		}
	}
	for _, selection := range s.config.Forge.Selections {
		if selected[selection.Resource] {
			config.Selections = append(config.Selections, selection)
		}
	}
	s.config.Forge = &config
	return nil
}
