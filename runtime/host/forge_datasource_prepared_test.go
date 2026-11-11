package host

import (
	"context"
	"encoding/json"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/viant/authz"
	"github.com/viant/authz/gating"
	"github.com/viant/authz/oauth"
	primitive "github.com/viant/forge/backend/provider/contract"
	identity "github.com/viant/forge/backend/resource"
	"github.com/viant/forge/backend/types"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type preparedDatasourcePolicyTrace struct {
	authz.Store
	mu        sync.Mutex
	resources []authz.Resource
}

func (s *preparedDatasourcePolicyTrace) Get(ctx context.Context, r authz.Resource) (authz.Document, error) {
	s.mu.Lock()
	s.resources = append(s.resources, r)
	s.mu.Unlock()
	return s.Store.Get(ctx, r)
}
func (s *preparedDatasourcePolicyTrace) reset() { s.mu.Lock(); s.resources = nil; s.mu.Unlock() }
func (s *preparedDatasourcePolicyTrace) snapshot() []authz.Resource {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]authz.Resource(nil), s.resources...)
}

type preparedDatasourceProofCounter struct {
	types.WindowTargetProof
	verifies atomic.Int64
}

func (p *preparedDatasourceProofCounter) Verify(ctx context.Context, pin identity.ResolvedResource, target types.WindowTarget, binding, token string) error {
	p.verifies.Add(1)
	return p.WindowTargetProof.Verify(ctx, pin, target, binding, token)
}

type preparedDatasourceFixture struct {
	service       *Service
	ctx           context.Context
	input         stockPublishedDatasourceInput
	entry         preparedStockDatasource
	trace         *preparedDatasourcePolicyTrace
	proofs        *preparedDatasourceProofCounter
	metadataTrace *preparedDatasourcePolicyTrace
}

func newPreparedDatasourceFixture(t *testing.T) preparedDatasourceFixture {
	t.Helper()
	reader := authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}
	fixture, err := newScopedHost(t, map[string]string{"records": unscopedRecordsDQL}, map[string]authz.Policy{"records": reader})
	require.NoError(t, err)
	service := fixture.service
	identityProvider := service.resourceAccess.Provider
	principals, ok := identityProvider.(gating.PrincipalResolver)
	require.True(t, ok)
	now := time.Now()
	claims := jwtlib.MapClaims{"iss": scopeTestIssuer, "aud": scopeTestAudience, "sub": "alice", "exp": now.Add(time.Minute).Unix(), "iat": now.Add(-time.Second).Unix(), "tenant": scopeTestTenant, "accountId": "21", "roles": []string{"reader"}}
	token, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims).SignedString(fixture.key)
	require.NoError(t, err)
	ctx := oauth.WithBearer(context.Background(), token)
	docs := []authz.Document{{Resource: authz.Resource{Kind: "component", ID: "records", Version: "1", Tenant: scopeTestTenant}, Revision: 1, Policies: map[string]authz.Policy{"describe": reader, "execute": reader}}}
	store, err := authz.NewStaticStore(docs)
	require.NoError(t, err)
	trace := &preparedDatasourcePolicyTrace{Store: store}
	service.resourceAccess.Store = trace
	ref := ComponentReference{Kind: "dynamic", ID: "records", Revision: "1", Method: http.MethodGet, Route: "/records"}
	manager, binding, err := service.componentRuntimeWithAuthority(ctx, ref, false)
	require.NoError(t, err)
	require.NoError(t, manager.Shutdown(context.Background()))
	// Exact producer representation is compiled before this fixture is exposed.
	definition, err := json.Marshal(map[string]any{"schemaVersion": 1, "dataSource": map[string]any{"id": "records", "backend": map[string]any{"kind": "datly", "ownership": "provider", "method": "datasources/fetch", "component": binding, "schemaFingerprint": binding.SchemaFingerprint}}})
	require.NoError(t, err)
	require.NoError(t, validateStockDatasourceDefinition(1, definition))
	entry := preparedStockDatasource{id: "records", definition: definition, fingerprint: identity.ContentFingerprint(definition), reference: ref, component: binding}
	service.config.Forge = &ForgeConfig{ProviderIdentity: "prepared-fixture"}
	uri, err := identity.ParseResourceURI("datasource://team/overview/records")
	require.NoError(t, err)
	configured := stockDatasourceEntry{id: "records", uri: uri, standalone: &ForgeDataSource{URI: uri.String()}}
	service.windowPrimitiveSource = &stockPrimitiveSource{datasources: map[string]stockDatasourceEntry{uri.String(): configured}}
	resource := authz.Resource{Kind: "datasource", ID: "team/overview/records", Version: identity.WorkingCandidate, Tenant: "*"}
	metadataStore, err := authz.NewStaticStore([]authz.Document{{Resource: resource, Revision: 1, Policies: map[string]authz.Policy{"execute": reader}}})
	require.NoError(t, err)
	selections, err := authz.NewStaticSelectionStore([]authz.SelectionDocument{{Resource: authz.ResourceFamily{Kind: resource.Kind, ID: resource.ID, Tenant: resource.Tenant}, Revision: 1, DefaultVersion: identity.WorkingCandidate}})
	require.NoError(t, err)
	metadataTrace := &preparedDatasourcePolicyTrace{Store: metadataStore}
	service.windowPrimitivePolicy = &stockWindowPolicy{service: service, tenant: "*", principals: principals, selector: &authz.Selector{Mappings: selections, Service: &authz.Service{Store: metadataTrace, Provider: identityProvider}}}
	service.windowPrimitiveSource.preparedDatasources.Store(&stockDatasourceGeneration{entries: map[string]preparedStockDatasource{"datasource://team/overview/records": entry}})
	signer, err := types.NewWindowTargetHMAC([]byte(strings.Repeat("p", 32)))
	require.NoError(t, err)
	proofs := &preparedDatasourceProofCounter{WindowTargetProof: signer}
	service.windowExecutionProof = proofs
	pin := identity.ResolvedResource{ProviderIdentity: "prepared-fixture", URI: "datasource://team/overview/records", ResourceCandidate: identity.ResourceCandidate{Kind: identity.WorkingCandidate, ContentFingerprint: entry.fingerprint}, AuthorityBinding: "original-producer-binding", ValidUntil: time.Now().Add(time.Minute)}
	decision, err := service.datasourceResolver(configured, "execute").Policy.SelectRevision(ctx, identity.ResourceRef{URI: pin.URI, Revision: "working"}, []identity.ResourceCandidate{pin.ResourceCandidate})
	require.NoError(t, err)
	pin.AuthorityBinding = decision.AuthorityBinding
	pin.ValidUntil = decision.ValidUntil
	get := primitive.GetResult{Resource: &primitive.ResourceState{}, ResolvedResource: &pin}
	require.NoError(t, service.signDatasourceExecutionProof(ctx, &get))
	trace.reset()
	metadataTrace.reset()
	return preparedDatasourceFixture{service, ctx, stockPublishedDatasourceInput{Resource: pin, ExecutionProof: get.ExecutionProof, DataSourceID: "records"}, entry, trace, proofs, metadataTrace}
}
func TestPreparedPublishedDatasourceExecutesOnlySelectedComponent(t *testing.T) {
	f := newPreparedDatasourceFixture(t)
	result, err := f.service.publishedDatasourceFetch(f.ctx, f.input)
	require.NoError(t, err)
	require.Contains(t, string(result), "alpha-101")
	require.Len(t, f.metadataTrace.snapshot(), 1, "select the exact datasource policy once without rereading its prepared definition")
	require.EqualValues(t, 1, f.proofs.verifies.Load(), "one proof verification per invocation")
	observed := f.trace.snapshot()
	require.NotEmpty(t, observed, "selected component data authorization must remain")
	for _, resource := range observed {
		require.NotEqual(t, "window", resource.Kind)
		if resource.Kind == "component" {
			require.Equal(t, "records", resource.ID)
		}
	}
	// Native component authorization still denies actual data, regardless of proof.
	deny := authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "not-a-reader"}}
	allow := authz.Policy{Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}
	store, err := authz.NewStaticStore([]authz.Document{{Resource: authz.Resource{Kind: "component", ID: "records", Version: "1", Tenant: scopeTestTenant}, Revision: 2, Policies: map[string]authz.Policy{"describe": allow, "execute": deny}}})
	require.NoError(t, err)
	f.service.resourceAccess.Store = store
	result, err = f.service.publishedDatasourceFetch(f.ctx, f.input)
	require.Error(t, err)
	require.Empty(t, result)
	require.ErrorIs(t, err, authz.ErrDenied)
}
func TestPreparedPublishedDatasourceRejectsChangedGenerationAndTamperedProof(t *testing.T) {
	f := newPreparedDatasourceFixture(t)
	tampered := f.input
	proof := *tampered.ExecutionProof
	proof.Token = "not-a-signature"
	tampered.ExecutionProof = &proof
	result, err := f.service.publishedDatasourceFetch(f.ctx, tampered)
	require.Error(t, err)
	require.Empty(t, result)
	require.Empty(t, f.trace.snapshot(), "bad proof must not execute native data")
	changed := f.entry
	var definition map[string]any
	require.NoError(t, json.Unmarshal(changed.definition, &definition))
	definition["dataSource"].(map[string]any)["title"] = "new generation"
	changed.definition, err = json.Marshal(definition)
	require.NoError(t, err)
	changed.fingerprint = identity.ContentFingerprint(changed.definition)
	f.service.windowPrimitiveSource.preparedDatasources.Store(&stockDatasourceGeneration{entries: map[string]preparedStockDatasource{f.input.Resource.URI: changed}})
	result, err = f.service.publishedDatasourceFetch(f.ctx, f.input)
	require.ErrorIs(t, err, identity.ErrResourceStale)
	require.Empty(t, result)
	require.Empty(t, f.trace.snapshot())
	fresh := f.input
	fresh.Resource.ContentFingerprint = changed.fingerprint
	get := primitive.GetResult{Resource: &primitive.ResourceState{}, ResolvedResource: &fresh.Resource}
	require.NoError(t, f.service.signDatasourceExecutionProof(f.ctx, &get))
	fresh.ExecutionProof = get.ExecutionProof
	result, err = f.service.publishedDatasourceFetch(f.ctx, fresh)
	require.NoError(t, err)
	require.Contains(t, string(result), "alpha-101")
	f.service.windowPrimitiveSource.invalidated.Store(true)
	result, err = f.service.publishedDatasourceFetch(f.ctx, fresh)
	require.ErrorIs(t, err, identity.ErrResourceStale)
	require.Empty(t, result)
}
func TestPreparedPublishedDatasourceConcurrentReadersUseCoherentGeneration(t *testing.T) {
	f := newPreparedDatasourceFixture(t)
	start := make(chan struct{})
	errs := make(chan error, 4)
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			result, err := f.service.publishedDatasourceFetch(f.ctx, f.input)
			if err == nil && !strings.Contains(string(result), "alpha-101") {
				err = identity.ErrResourceDenied
			}
			if err != nil && err != identity.ErrResourceStale {
				errs <- err
			}
		}()
	}
	close(start)
	changed := f.entry
	var definition map[string]any
	require.NoError(t, json.Unmarshal(changed.definition, &definition))
	definition["dataSource"].(map[string]any)["title"] = "concurrent replacement"
	var err error
	changed.definition, err = json.Marshal(definition)
	require.NoError(t, err)
	changed.fingerprint = identity.ContentFingerprint(changed.definition)
	f.service.windowPrimitiveSource.preparedDatasources.Store(&stockDatasourceGeneration{entries: map[string]preparedStockDatasource{f.input.Resource.URI: changed}})
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	for _, resource := range f.trace.snapshot() {
		require.NotEqual(t, "window", resource.Kind)
		if resource.Kind == "component" {
			require.Equal(t, "records", resource.ID)
		}
	}
}
