package host

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	primitive "github.com/viant/forge/backend/provider/contract"
	identity "github.com/viant/forge/backend/resource"
	"github.com/viant/forge/backend/types"
	windowprotocol "github.com/viant/forge/backend/window/contract"
)

type stockDatasourceEntry struct {
	owner stockWindowEntry
	id    string
	uri   identity.ResourceURI
}
type stockDatasourceDefinition struct {
	SchemaVersion int64                      `json:"schemaVersion"`
	DataSource    *windowprotocol.DataSource `json:"dataSource"`
}

func stockDatasourceURI(window identity.ResourceURI, id string) identity.ResourceURI {
	return identity.ResourceURI{Kind: "datasource", Namespace: window.Namespace, Name: window.Name + "/" + id}
}

type stockDatasourceSource struct {
	canonical *stockPrimitiveSource
	entry     stockDatasourceEntry
}

func (s stockDatasourceSource) bytes(ctx context.Context, uri identity.ResourceURI) (json.RawMessage, error) {
	if uri != s.entry.uri {
		return nil, identity.ErrResourceDenied
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.canonical.invalidated.Load() {
		return nil, identity.ErrResourceStale
	}
	generation := s.canonical.preparedDatasources.Load()
	if generation == nil {
		return nil, identity.ErrResourceStale
	}
	entry, exists := generation.entries[uri.String()]
	if !exists {
		return nil, identity.ErrResourceDenied
	}
	return append(json.RawMessage(nil), entry.definition...), nil
}
func (s stockDatasourceSource) Candidates(ctx context.Context, uri identity.ResourceURI) ([]identity.ResourceCandidate, error) {
	raw, err := s.bytes(ctx, uri)
	if err != nil {
		return nil, err
	}
	return []identity.ResourceCandidate{{Kind: identity.WorkingCandidate, ContentFingerprint: identity.ContentFingerprint(raw)}}, nil
}
func (s stockDatasourceSource) ReadCandidate(ctx context.Context, uri identity.ResourceURI, candidate identity.ResourceCandidate) (json.RawMessage, error) {
	if candidate.Kind != identity.WorkingCandidate || candidate.Revision != "" {
		return nil, identity.ErrResourceDenied
	}
	return s.bytes(ctx, uri)
}
func (s stockDatasourceSource) localDefinitionCurrent(ctx context.Context, uri identity.ResourceURI, candidate identity.ResourceCandidate) error {
	raw, err := s.bytes(ctx, uri)
	if err != nil {
		return err
	}
	if identity.ContentFingerprint(raw) != candidate.ContentFingerprint {
		return identity.ErrResourceStale
	}
	return nil
}

type stockDatasourcePolicy struct {
	base   *stockWindowPolicy
	entry  stockDatasourceEntry
	action string
}

func (p stockDatasourcePolicy) SelectRevision(ctx context.Context, ref identity.ResourceRef, candidates []identity.ResourceCandidate) (identity.ResourceDecision, error) {
	if ref.URI != p.entry.uri.String() || ref.Revision != "" && ref.Revision != identity.WorkingCandidate {
		return identity.ResourceDecision{}, identity.ErrResourceDenied
	}
	selected, err := p.base.selectWindow(ctx, p.entry.owner, p.action)
	if err != nil || selected.Resource.Version != identity.WorkingCandidate {
		return identity.ResourceDecision{}, identity.ErrResourceDenied
	}
	principal, err := p.base.principal(ctx)
	if err != nil {
		return identity.ResourceDecision{}, err
	}
	for _, candidate := range candidates {
		if candidate.Kind == identity.WorkingCandidate && candidate.Revision == "" {
			return identity.ResourceDecision{Candidate: candidate, AuthorityBinding: identity.ContentFingerprint([]byte(stockAuthorityBinding(principal) + "\x00" + p.entry.uri.String() + "\x00" + p.entry.owner.config.URI)), ValidUntil: principal.Facts.ValidUntil}, nil
		}
	}
	return identity.ResourceDecision{}, identity.ErrResourceDenied
}
func validateStockDatasourceDefinition(version int64, raw json.RawMessage) error {
	var definition stockDatasourceDefinition
	if version != 1 || json.Unmarshal(raw, &definition) != nil || definition.SchemaVersion != 1 || definition.DataSource == nil || definition.DataSource.ID == "" || definition.DataSource.Backend == nil {
		return identity.ErrResourceDenied
	}
	backend := definition.DataSource.Backend
	if backend.Kind != "datly" || backend.Ownership != "provider" || backend.Service != "" || backend.Method != "datasources/fetch" || backend.Component == nil || backend.SchemaFingerprint != backend.Component.SchemaFingerprint {
		return identity.ErrResourceDenied
	}
	pin := backend.Component
	if pin.Kind != "dynamic" && pin.Kind != "linked" || pin.ID == "" || pin.Revision == "" || pin.Revision == "working" || pin.Revision == "active" || pin.Revision == "latest" || !(identity.ResourceCandidate{Kind: identity.WorkingCandidate, ContentFingerprint: pin.ContentFingerprint}).Valid() || !(identity.ResourceCandidate{Kind: identity.WorkingCandidate, ContentFingerprint: pin.SchemaFingerprint}).Valid() {
		return identity.ErrResourceDenied
	}
	return nil
}

const studioDatasourceExecutionDomain = "studio.datasource.v1:"

type stockPublishedDatasourceInput struct {
	DataSourceID   string                    `json:"dataSourceId,omitempty"`
	Resource       identity.ResolvedResource `json:"resource"`
	ExecutionProof *primitive.ExecutionProof `json:"executionProof"`
	Inputs         map[string]any            `json:"inputs,omitempty"`
}

func (s *Service) datasourceResolver(entry stockDatasourceEntry, action string) *identity.ResourceResolver {
	return &identity.ResourceResolver{ProviderIdentity: s.config.Forge.ProviderIdentity, Source: stockDatasourceSource{canonical: s.windowPrimitiveSource, entry: entry}, Policy: stockDatasourcePolicy{base: s.windowPrimitivePolicy, entry: entry, action: action}}
}
func (s *Service) signDatasourceExecutionProof(ctx context.Context, get *primitive.GetResult) error {
	if get == nil || get.Resource == nil || get.ResolvedResource == nil {
		return identity.ErrResourceDenied
	}
	binding := studioDatasourceExecutionDomain + get.ResolvedResource.ContentFingerprint
	token, err := s.windowExecutionProof.Sign(ctx, *get.ResolvedResource, types.WindowTarget{}, binding)
	if err != nil {
		return err
	}
	get.ExecutionProof = &primitive.ExecutionProof{Resource: *get.ResolvedResource, Binding: binding, Token: token}
	return nil
}
func (s *Service) publishedDatasourceFetch(ctx context.Context, in stockPublishedDatasourceInput) (json.RawMessage, error) {
	if s.windowPrimitiveSource == nil || in.Resource.ProviderIdentity != s.config.Forge.ProviderIdentity || in.ExecutionProof == nil {
		return nil, identity.ErrResourceDenied
	}
	if s.windowPrimitiveSource.invalidated.Load() {
		return nil, identity.ErrResourceStale
	}
	generation := s.windowPrimitiveSource.preparedDatasources.Load()
	if generation == nil {
		return nil, identity.ErrResourceStale
	}
	entry, ok := generation.entries[in.Resource.URI]
	if !ok || in.DataSourceID != "" && in.DataSourceID != entry.id {
		return nil, identity.ErrResourceDenied
	}
	if in.Resource.ContentFingerprint != entry.fingerprint {
		return nil, identity.ErrResourceStale
	}
	pin := in.Resource
	verify := func() error {
		proof := in.ExecutionProof
		if proof.Binding != studioDatasourceExecutionDomain+pin.ContentFingerprint || proof.Token == "" || proof.Resource.ProviderIdentity != pin.ProviderIdentity || proof.Resource.URI != pin.URI || proof.Resource.ResourceCandidate != pin.ResourceCandidate || proof.Resource.AuthorityBinding != pin.AuthorityBinding || pin.ValidUntil.After(proof.Resource.ValidUntil) || !pin.ValidUntil.After(time.Now()) || !proof.Resource.ValidUntil.After(time.Now()) {
			return identity.ErrResourceDenied
		}
		return s.windowExecutionProof.Verify(ctx, proof.Resource, types.WindowTarget{}, proof.Binding, proof.Token)
	}
	if err := verify(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithDeadline(ctx, pin.ValidUntil)
	defer cancel()
	// Publication captured the exact descriptor/component contract once. Runtime
	// performs one provenance check above and delegates only this selected native
	// component's actual data authorization/execution to the producer runtime.
	return s.ExecuteComponentJSON(ctx, entry.reference, entry.component, in.Inputs)
}
func (s *Service) datasourceEntry(uri string) (stockDatasourceEntry, error) {
	if s.windowPrimitiveSource == nil {
		return stockDatasourceEntry{}, identity.ErrResourceDenied
	}
	entry, ok := s.windowPrimitiveSource.datasources[uri]
	if !ok {
		return entry, fmt.Errorf("configured datasource unavailable")
	}
	return entry, nil
}
