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
	raw, err := s.canonical.bytes(ctx, s.entry.owner.uri)
	if err != nil {
		return nil, err
	}
	variant, err := types.SelectWindowResourceWithReferences(raw, nil)
	if err != nil {
		return nil, err
	}
	var source windowprotocol.DataSource
	if json.Unmarshal(variant.DataSources[s.entry.id], &source) != nil || source.ID != s.entry.id {
		return nil, identity.ErrResourceDenied
	}
	definition := stockDatasourceDefinition{SchemaVersion: 1, DataSource: &source}
	result, err := json.Marshal(definition)
	if err != nil {
		return nil, err
	}
	if err = validateStockDatasourceDefinition(1, result); err != nil {
		return nil, err
	}
	return result, nil
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
	entry, ok := s.windowPrimitiveSource.datasources[in.Resource.URI]
	if !ok || in.DataSourceID != "" && in.DataSourceID != entry.id {
		return nil, identity.ErrResourceDenied
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
	resolver := s.datasourceResolver(entry, "execute")
	approved, current, err := resolver.ReadResolved(ctx, pin)
	if err != nil {
		return nil, err
	}
	if current.ValidUntil.Before(pin.ValidUntil) {
		pin.ValidUntil = current.ValidUntil
		var narrowCancel context.CancelFunc
		ctx, narrowCancel = context.WithDeadline(ctx, pin.ValidUntil)
		defer narrowCancel()
	}
	var definition stockDatasourceDefinition
	if json.Unmarshal(approved, &definition) != nil || definition.DataSource == nil || definition.DataSource.Backend == nil || definition.DataSource.Backend.Component == nil {
		return nil, identity.ErrResourceDenied
	}
	ref, ok := entry.owner.config.Components[entry.id]
	if !ok {
		return nil, identity.ErrResourceDenied
	}
	if err := verify(); err != nil {
		return nil, err
	}
	result, err := s.ExecuteComponentJSON(ctx, ref, *definition.DataSource.Backend.Component, in.Inputs)
	if err != nil {
		return nil, err
	}
	final, finalPin, err := resolver.ReadResolved(ctx, pin)
	if err != nil || string(final) != string(approved) {
		return nil, identity.ErrResourceStale
	}
	if finalPin.ValidUntil.Before(pin.ValidUntil) {
		pin.ValidUntil = finalPin.ValidUntil
	}
	if err = verify(); err != nil {
		return nil, err
	}
	if _, err := s.windowPrimitivePolicy.principal(ctx); err != nil {
		return nil, err
	}
	if err = verify(); err != nil {
		return nil, err
	}
	return result, nil
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
