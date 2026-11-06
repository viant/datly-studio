package host

import (
	"context"
	"encoding/json"
	identity "github.com/viant/forge/backend/resource"
	resources "github.com/viant/forge/backend/resourcehost"
	"github.com/viant/forge/backend/types"
	windowprotocol "github.com/viant/forge/backend/window/contract"
)

// Published atomically only after all producer-owned datasource definitions
// validate at preload/publication. Readers capture one immutable generation.
type stockDatasourceGeneration struct {
	entries map[string]preparedStockDatasource
}
type preparedStockDatasource struct {
	id          string
	definition  json.RawMessage
	fingerprint string
	reference   ComponentReference
	component   windowprotocol.ComponentBinding
}

func compileStockDatasourceGeneration(ctx context.Context, snapshot *resources.NativeAssetSnapshot, windows map[string]stockWindowEntry, standalone map[string]ForgeDataSource) (*stockDatasourceGeneration, error) {
	generation := &stockDatasourceGeneration{entries: map[string]preparedStockDatasource{}}
	for _, window := range windows {
		if !window.config.PublishDataSources {
			continue
		}
		candidates, err := snapshot.Candidates(ctx, window.uri)
		if err != nil {
			return nil, err
		}
		if len(candidates) != 1 {
			return nil, identity.ErrResourceStale
		}
		raw, err := snapshot.ReadCandidate(ctx, window.uri, candidates[0])
		if err != nil {
			return nil, err
		}
		variant, err := types.SelectWindowResourceWithReferences(raw, nil)
		if err != nil {
			return nil, err
		}
		for id, reference := range window.config.Components {
			var descriptor windowprotocol.DataSource
			if err = json.Unmarshal(variant.DataSources[id], &descriptor); err != nil {
				return nil, err
			}
			definition, err := json.Marshal(stockDatasourceDefinition{SchemaVersion: 1, DataSource: &descriptor})
			if err != nil {
				return nil, err
			}
			if err = validateStockDatasourceDefinition(1, definition); err != nil {
				return nil, err
			}
			uri := stockDatasourceURI(window.uri, id).String()
			if _, exists := generation.entries[uri]; exists {
				return nil, identity.ErrResourceDenied
			}
			generation.entries[uri] = preparedStockDatasource{id: id, definition: definition, fingerprint: identity.ContentFingerprint(definition), reference: reference, component: *descriptor.Backend.Component}
		}
	}
	for _, configured := range standalone {
		uri, err := identity.ParseResourceURI(configured.URI)
		if err != nil {
			return nil, err
		}
		candidates, err := snapshot.Candidates(ctx, uri)
		if err != nil || len(candidates) != 1 {
			return nil, identity.ErrResourceStale
		}
		raw, err := snapshot.ReadCandidate(ctx, uri, candidates[0])
		if err != nil {
			return nil, err
		}
		var definition stockDatasourceDefinition
		if err = json.Unmarshal(raw, &definition); err != nil {
			return nil, err
		}
		if err = validateStockDatasourceDefinition(1, raw); err != nil {
			return nil, err
		}
		if _, exists := generation.entries[configured.URI]; exists {
			return nil, identity.ErrResourceDenied
		}
		generation.entries[configured.URI] = preparedStockDatasource{id: definition.DataSource.ID, definition: raw, fingerprint: identity.ContentFingerprint(raw), reference: configured.Component, component: *definition.DataSource.Backend.Component}
	}
	return generation, nil
}
