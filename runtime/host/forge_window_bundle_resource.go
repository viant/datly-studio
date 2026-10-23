package host

import (
	"bytes"
	"context"
	"encoding/json"
	"sort"
	"time"

	primitive "github.com/viant/forge/backend/provider/contract"
	identity "github.com/viant/forge/backend/resource"
	"github.com/viant/forge/backend/types"
	windowprotocol "github.com/viant/forge/backend/window/contract"
	"github.com/viant/jsonrpc"
	"github.com/viant/mcp-protocol/schema"
)

// stockWindowClosure is content only. It contains no principal, bearer,
// execution proof, permission or lease. A hash is an integrity check, never an
// authorization grant. The MCP read response itself remains private and TTL 0.
type stockWindowClosure struct {
	SchemaVersion int                       `json:"schemaVersion"`
	Format        string                    `json:"format"`
	WindowURI     string                    `json:"windowUri"`
	ContentHash   string                    `json:"contentHash"`
	Complete      bool                      `json:"complete"`
	Entries       []stockWindowClosureEntry `json:"entries"`
}

type stockWindowClosureEntry struct {
	ID                 string `json:"id,omitempty"`
	URI                string `json:"uri"`
	Revision           string `json:"revision"`
	ContentFingerprint string `json:"contentFingerprint"`
	DefinitionText     string `json:"definitionText"`
}

func (h *stockResourceHandler) readWindowBundleResource(ctx context.Context, requested identity.ResourceURI) (*schema.ReadResourceResult, *jsonrpc.Error) {
	raw, err := h.stockWindowClosure(ctx, requested)
	if err != nil {
		return nil, jsonrpc.NewInvalidRequest("configured window closure denied or unavailable", nil)
	}
	mime := "application/json"
	return &schema.ReadResourceResult{
		CacheScope: schema.ReadResourceResultCacheScopePrivate,
		ResultType: schema.ResultTypeComplete,
		TtlMs:      0,
		Contents:   []schema.ReadResourceResultContentsElem{{Uri: requested.String(), MimeType: &mime, Text: string(raw)}},
	}, nil
}

type compiledStockWindowClosure struct {
	windowFingerprint string
	content           []byte
}

func (h *stockResourceHandler) stockWindowClosure(ctx context.Context, requested identity.ResourceURI) ([]byte, error) {
	if h == nil || h.host == nil || h.host.windowPrimitives == nil || h.host.windowPrimitiveSource == nil || requested.Kind != "bundle" || ctx == nil || ctx.Err() != nil {
		return nil, identity.ErrResourceDenied
	}
	window := identity.ResourceURI{Kind: "window", Namespace: requested.Namespace, Name: requested.Name}
	entry, ok := h.host.windowPrimitiveSource.original.entries[window.String()]
	compiled, exists := h.host.windowPrimitiveSource.closures[window.String()]
	if !ok || !entry.config.PublishDataSources || !exists {
		return nil, identity.ErrResourceDenied
	}
	get := func() (*primitive.GetResult, []byte, error) {
		selected, err := h.host.windowPrimitives.Get(ctx, "window", primitive.GetRequest{URI: window.String(), Revision: identity.WorkingCandidate})
		if err != nil || selected == nil || selected.Resource == nil || selected.ResolvedResource == nil || !selected.ResolvedResource.ValidUntil.After(time.Now()) {
			return nil, nil, identity.ErrResourceDenied
		}
		raw := selected.Resource.DefinitionBytes
		if len(raw) == 0 {
			raw = selected.Resource.Definition
		}
		if !json.Valid(raw) || identity.ContentFingerprint(raw) != selected.ResolvedResource.ContentFingerprint {
			return nil, nil, identity.ErrResourceStale
		}
		return selected, raw, nil
	}
	parent, windowBytes, err := get()
	if err != nil {
		return nil, err
	}
	if parent.ResolvedResource.ContentFingerprint != compiled.windowFingerprint || identity.ContentFingerprint(windowBytes) != compiled.windowFingerprint {
		return nil, identity.ErrResourceStale
	}
	// The same current parent Get authorizes the window and every configured
	// component. The final parent read fences identity, permission and source
	// changes after the immutable snapshot bytes were selected.
	final, finalBytes, err := get()
	if err != nil || !bytes.Equal(finalBytes, windowBytes) || final.ResolvedResource.ResourceCandidate != parent.ResolvedResource.ResourceCandidate || final.ResolvedResource.AuthorityBinding != parent.ResolvedResource.AuthorityBinding {
		return nil, identity.ErrResourceStale
	}
	return bytes.Clone(compiled.content), nil
}

// compileStockWindowClosure runs only while the stock snapshot is being
// activated. It accepts trusted, already compiled window bytes and validates
// the entire declared datasource closure before any request can read it.
func compileStockWindowClosure(windowBytes []byte, entry stockWindowEntry, providerIdentity string) (compiledStockWindowClosure, error) {
	var manifest types.WindowResourceEnvelope
	// The stock loader currently materializes one byte-identical variant for
	// default and web/desktop. Future target-varying formats need explicit
	// target-aware datasource selection before this cache can serve them.
	if json.Unmarshal(windowBytes, &manifest) != nil || manifest.SchemaVersion != 3 || manifest.Format != types.WindowReferencesFormat || len(manifest.Variants) != 1 || manifest.Validate() != nil {
		return compiledStockWindowClosure{}, identity.ErrResourceStale
	}
	refs, err := stockClosureReferences(&manifest, entry, providerIdentity)
	if err != nil {
		return compiledStockWindowClosure{}, err
	}
	ids := make([]string, 0, len(refs))
	for id := range refs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	parentHash := identity.ContentFingerprint(windowBytes)
	entries := []stockWindowClosureEntry{{URI: entry.uri.String(), Revision: identity.WorkingCandidate, ContentFingerprint: parentHash, DefinitionText: string(windowBytes)}}
	for _, id := range ids {
		ref := refs[id]
		var raw []byte
		for _, variant := range manifest.Variants {
			descriptor := variant.DataSources[id]
			if len(descriptor) == 0 {
				continue
			}
			var source windowprotocol.DataSource
			if json.Unmarshal(descriptor, &source) != nil || source.ID != id {
				return compiledStockWindowClosure{}, identity.ErrResourceStale
			}
			candidate, err := json.Marshal(stockDatasourceDefinition{SchemaVersion: 1, DataSource: &source})
			if err != nil || validateStockDatasourceDefinition(1, candidate) != nil || identity.ContentFingerprint(candidate) != ref.ContentFingerprint || raw != nil && !bytes.Equal(raw, candidate) {
				return compiledStockWindowClosure{}, identity.ErrResourceStale
			}
			raw = candidate
		}
		if len(raw) == 0 {
			return compiledStockWindowClosure{}, identity.ErrResourceStale
		}
		entries = append(entries, stockWindowClosureEntry{ID: id, URI: ref.Resource.URI, Revision: identity.WorkingCandidate, ContentFingerprint: ref.ContentFingerprint, DefinitionText: string(raw)})
	}
	content, err := json.Marshal(entries)
	if err != nil {
		return compiledStockWindowClosure{}, err
	}
	encoded, err := json.Marshal(stockWindowClosure{SchemaVersion: 1, Format: "window.closure", WindowURI: entry.uri.String(), ContentHash: identity.ContentFingerprint(content), Complete: true, Entries: entries})
	if err != nil {
		return compiledStockWindowClosure{}, err
	}
	return compiledStockWindowClosure{windowFingerprint: parentHash, content: encoded}, nil
}

func stockClosureReferences(manifest *types.WindowResourceEnvelope, entry stockWindowEntry, providerIdentity string) (map[string]primitive.DataSourceReference, error) {
	if manifest == nil || providerIdentity == "" {
		return nil, identity.ErrResourceDenied
	}
	refs := map[string]primitive.DataSourceReference{}
	for _, variant := range manifest.Variants {
		for id, ref := range variant.DataSourceResources {
			if _, configured := entry.config.Components[id]; !configured || ref.ProviderIdentity != providerIdentity || ref.Resource.URI != stockDatasourceURI(entry.uri, id).String() || ref.Resource.Revision != identity.WorkingCandidate || ref.ContentFingerprint == "" {
				return nil, identity.ErrResourceDenied
			}
			if previous, exists := refs[id]; exists && previous != ref {
				return nil, identity.ErrResourceStale
			}
			refs[id] = ref
		}
	}
	// Published datasource mode binds every configured component to a resource;
	// an unknown or omitted child must never produce a partial closure.
	if len(refs) != len(entry.config.Components) {
		return nil, identity.ErrResourceStale
	}
	return refs, nil
}
