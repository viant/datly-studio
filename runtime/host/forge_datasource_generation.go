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

func compileStockDatasourceGeneration(ctx context.Context, snapshot *resources.NativeAssetSnapshot, windows map[string]stockWindowEntry) (*stockDatasourceGeneration, error) {
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
	return generation, nil
}
