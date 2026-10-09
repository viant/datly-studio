package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	primitive "github.com/viant/agently-core/protocol/primitive"
	identity "github.com/viant/agently-core/protocol/resource"
	coreresource "github.com/viant/agently-core/service/resource"
	"github.com/viant/authz"
	studiors "github.com/viant/datly-studio/runtime/resources"
	"strconv"
	"time"
)

func (s *stockPrimitiveSource) preload() error {
	// Startup observes static content only; no actor or request values enter
	// compilation. Managers are closed immediately, never retained in the cache.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	s.components = map[ComponentReference]string{}
	compiled := map[ComponentReference]stockComponentPin{}
	definitions := make([]coreresource.NativeSnapshotDefinition, 0, len(s.original.entries))
	assetPaths := []string{}
	seenPaths := map[string]bool{}
	for _, entry := range s.original.entries {
		entry := entry
		if !seenPaths[entry.config.DefinitionPath] {
			assetPaths = append(assetPaths, entry.config.DefinitionPath)
			seenPaths[entry.config.DefinitionPath] = true
		}
		definitions = append(definitions, coreresource.NativeSnapshotDefinition{URI: entry.config.URI, Title: entry.uri.Name, FormatVersion: 2, File: entry.config.DefinitionPath, Load: func(ctx context.Context, reader *coreresource.ExtensionReader, file string) (json.RawMessage, error) {
			raw, err := reader.ReadFile(ctx, file)
			if err != nil {
				return nil, err
			}
			definition, err := decodeStockWindowFile(raw)
			if err != nil || validateStockWindow(entry, definition) != nil {
				return nil, fmt.Errorf("configured window is invalid")
			}
			pins := map[string]stockComponentPin{}
			for id, ref := range entry.config.Components {
				before, err := s.original.service.primitiveComponentSource(ctx, ref)
				if err != nil {
					return nil, err
				}
				pin, found := compiled[ref]
				if !found {
					manager, binding, err := s.original.service.componentRuntimeWithAuthority(ctx, ref, false)
					if err != nil {
						return nil, err
					}
					if err := manager.Shutdown(ctx); err != nil {
						return nil, err
					}
					s.metadataCompiles.Add(1)
					if ref.Kind == "dynamic" && binding.ContentFingerprint != before {
						return nil, identity.ErrResourceStale
					}
					pin = stockComponentPin{Reference: ref, Binding: binding}
					compiled[ref] = pin
				}
				after, err := s.original.service.primitiveComponentSource(ctx, ref)
				if err != nil || before != after {
					return nil, identity.ErrResourceStale
				}
				if original, exists := s.components[ref]; exists && original != after {
					return nil, identity.ErrResourceStale
				}
				s.components[ref], pins[id] = after, pin
			}
			return s.materialize(stockWindowBundle{URI: entry.config.URI, Source: raw, Components: pins}, entry)
		}})
	}
	snapshot, err := coreresource.NewNativeAssetSnapshot(ctx, s.original.root, definitions, coreresource.NativeSnapshotOptions{AssetPaths: assetPaths, ImmutableUntilRestart: true})
	if err != nil {
		return err
	}
	s.snapshot = snapshot
	return nil
}

func (s *Service) primitiveComponentSource(ctx context.Context, ref ComponentReference) (string, error) {
	if ref.Kind == "linked" {
		for _, source := range s.config.LinkedComponents {
			if source.ID == ref.ID && source.Revision == ref.Revision {
				raw, err := json.Marshal([]string{ref.ID, ref.Revision, source.ArtifactFingerprint})
				if err != nil {
					return "", err
				}
				return identity.ContentFingerprint(raw), nil
			}
		}
		return "", identity.ErrResourceStale
	}
	version, err := strconv.Atoi(ref.Revision)
	if err != nil {
		return "", err
	}
	definition, err := s.exactDefinition(ctx, ref.ID, version)
	if err != nil {
		return "", err
	}
	resources, err := s.metadataResources.Load(ctx, []studiors.Version{{ReportID: ref.ID, VersionNo: version}})
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(struct {
		ID, Scope, Name, Connector, Driver, DSN, Secret, DQL, Resources string
		Version                                                         int
	}{definition.reportID, definition.scope, definition.name, definition.connector, definition.driver, definition.dsn, definition.secretRef, definition.dql, resources.Fingerprints[studiors.Version{ReportID: ref.ID, VersionNo: version}], definition.versionNo})
	if err != nil {
		return "", err
	}
	return identity.ContentFingerprint(raw), nil
}

func (s *Service) initPrimitiveMetadata() error {
	definitions, err := newExactDefinitionReader(s.studio)
	if err != nil {
		return err
	}
	resources, err := studiors.NewLoader(s.studio)
	if err != nil {
		_ = definitions.runtime.Shutdown(context.Background())
		return err
	}
	s.metadataDefinitions, s.metadataResources = definitions, resources
	return nil
}
func (s *Service) closePrimitiveMetadata(ctx context.Context) error {
	var result error
	if s.metadataDefinitions != nil {
		result = errors.Join(result, s.metadataDefinitions.runtime.Shutdown(ctx))
	}
	if s.metadataResources != nil {
		result = errors.Join(result, s.metadataResources.Close(ctx))
	}
	return result
}

func (s *stockPrimitiveSource) check(ctx context.Context, uri identity.ResourceURI) error {
	if s.invalidated.Load() {
		return identity.ErrResourceStale
	}
	entry, ok := s.original.entries[uri.String()]
	if !ok {
		return identity.ErrResourceDenied
	}
	for _, ref := range entry.config.Components {
		fingerprint, err := s.original.service.primitiveComponentSource(ctx, ref)
		if err != nil {
			return err
		}
		if fingerprint != s.components[ref] {
			s.invalidated.Store(true)
			return identity.ErrResourceStale
		}
	}
	return ctx.Err()
}

func (s *stockPrimitiveSource) authorizeComponents(ctx context.Context, uri identity.ResourceURI) error {
	entry, ok := s.original.entries[uri.String()]
	if !ok {
		return identity.ErrResourceDenied
	}
	for _, ref := range entry.config.Components {
		service := s.original.service
		if ref.Kind == "dynamic" {
			version, _ := strconv.Atoi(ref.Revision)
			service = service.publishedAuthorizer(map[string]int{ref.ID: version})
		}
		if err := service.authorizeNamespace(ctx); err != nil {
			return err
		}
		if err := service.authorizeResourcePolicy(ctx, authz.Resource{Kind: "component", ID: ref.ID, Version: ref.Revision, Tenant: service.config.Access.Tenant}, "describe"); err != nil {
			return err
		}
	}
	return nil
}

func (s *stockPrimitiveSource) index(ctx context.Context) ([]primitive.ResourceState, error) {
	if s.invalidated.Load() {
		return nil, identity.ErrResourceStale
	}
	// Several windows can share one exact native component. Observe each
	// trusted source once per index read, without per-window compilation.
	for ref, original := range s.components {
		current, err := s.original.service.primitiveComponentSource(ctx, ref)
		if err != nil {
			return nil, err
		}
		if current != original {
			s.invalidated.Store(true)
			return nil, identity.ErrResourceStale
		}
	}
	return s.snapshot.WindowIndex(ctx)
}
