package host

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"sync"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/internal/namespacepolicy"
	discovery "github.com/viant/datly-studio/studio/resource_policy/discovery"
	"github.com/viant/datly/exec"
)

// catalogPolicyGraph contains policy metadata only. The Authz service still
// computes discover and describe separately for the authenticated caller.
type catalogPolicyGraph struct {
	documents  map[authz.Resource]authz.Document
	invalid    map[authz.Resource]bool
	components map[string]bool // value records whether any policy version exists
}

// Each published namespace runtime owns one index. A fresh database read
// fences changes by other writers; equal rows reuse the prepared documents.
type catalogPolicyIndex struct {
	mu                sync.Mutex
	namespace, tenant string
	rows              []*discovery.PolicyHead
	graph             *catalogPolicyGraph
}

func sameCatalogHead(a, b *discovery.PolicyHead) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.ResourceKind != b.ResourceKind || a.ResourceId != b.ResourceId || a.ResourceVersion != b.ResourceVersion || a.Revision != b.Revision {
		return false
	}
	if a.PoliciesJson == nil || b.PoliciesJson == nil {
		return a.PoliciesJson == b.PoliciesJson
	}
	return *a.PoliciesJson == *b.PoliciesJson
}

func (s *publishedDefinitionStore) readCatalogPolicyGraph(ctx context.Context, tenant, namespace string) (*catalogPolicyGraph, error) {
	input := &discovery.Input{Tenant: tenant, NamespaceId: namespace, Has: &discovery.InputHas{Tenant: true, NamespaceId: true}}
	value, err := s.runtime.InvokeComponent(ctx, exec.ComponentRequest{Target: s.policyCatalog, Input: input})
	if err != nil {
		return nil, err
	}
	output, ok := value.(*discovery.Output)
	if !ok || output == nil {
		return nil, errors.New("policy catalog returned an invalid result")
	}
	idx := &s.discoveryIndex
	idx.mu.Lock()
	defer idx.mu.Unlock()
	if idx.graph != nil && idx.namespace == namespace && idx.tenant == tenant && slices.EqualFunc(idx.rows, output.Heads, sameCatalogHead) {
		return idx.graph, nil
	}
	graph, err := newCatalogPolicyGraph(tenant, namespace, output.Heads)
	if err != nil {
		return nil, err
	}
	idx.namespace, idx.tenant, idx.rows, idx.graph = namespace, tenant, output.Heads, graph
	return graph, nil
}

func newCatalogPolicyGraph(tenant, namespace string, rows []*discovery.PolicyHead) (*catalogPolicyGraph, error) {
	g := &catalogPolicyGraph{documents: make(map[authz.Resource]authz.Document), invalid: make(map[authz.Resource]bool), components: make(map[string]bool)}
	hasNamespace := false
	for _, r := range rows {
		if r == nil || r.ResourceId == "" {
			return nil, authz.ErrUnavailable
		}
		switch r.ResourceKind {
		case "namespace":
			if r.ResourceId != namespace {
				return nil, authz.ErrUnavailable
			}
			hasNamespace = true
		case "component":
			g.components[r.ResourceId] = g.components[r.ResourceId] || r.ResourceVersion != "" || r.Revision != 0 || r.PoliciesJson != nil
		default:
			return nil, authz.ErrUnavailable
		}
		if r.ResourceVersion == "" && r.Revision == 0 && r.PoliciesJson == nil {
			continue
		}
		key := authz.Resource{Tenant: tenant, Kind: r.ResourceKind, ID: r.ResourceId, Version: r.ResourceVersion}
		doc := authz.Document{Resource: key, Revision: r.Revision}
		if r.ResourceVersion == "" || r.Revision <= 0 || r.PoliciesJson == nil || json.Unmarshal([]byte(*r.PoliciesJson), &doc.Policies) != nil {
			g.invalid[key] = true
			continue
		}
		g.documents[key] = doc
	}
	if !hasNamespace {
		return nil, authz.ErrDenied
	}
	return g, nil
}

func (g *catalogPolicyGraph) get(r authz.Resource) (authz.Document, error) {
	if g.invalid[r] {
		return authz.Document{}, authz.ErrUnavailable
	}
	doc, ok := g.documents[r]
	if !ok {
		return authz.Document{}, sql.ErrNoRows
	}
	return doc, nil
}

// Namespace inheritance and publication defaults remain owned by their existing
// stores. The graph replaces their individual lookups only for a discovery read.
type discoveryDocuments struct {
	authz.Store
	graph             *catalogPolicyGraph
	tenant, namespace string
}

func (d *discoveryDocuments) Get(ctx context.Context, key authz.Resource) (authz.Document, error) {
	if err := ctx.Err(); err != nil {
		return authz.Document{}, err
	}
	if key.Tenant == d.tenant {
		if key.Kind == "namespace" && key.ID == d.namespace {
			return d.graph.get(key)
		}
		if key.Kind == "component" {
			if _, known := d.graph.components[key.ID]; known {
				return d.graph.get(key)
			}
		}
	}
	return d.Store.Get(ctx, key)
}

// One catalog read has one authenticated identity. Authz still checks the
// verified facts' expiry and evaluates each resource/action separately.
type catalogFacts func() (authz.Facts, error)

func (f catalogFacts) Resolve(ctx context.Context) (authz.Facts, error) {
	if err := ctx.Err(); err != nil {
		return authz.Facts{}, err
	}
	return f()
}

type discoveryRequestKey struct{}
type discoveryRequest struct {
	once   sync.Once
	access *authz.Service
	err    error
}

func withDiscoveryRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), discoveryRequestKey{}, new(discoveryRequest))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Service) discoveryAccess(ctx context.Context, request *discoveryRequest) (*authz.Service, error) {
	request.once.Do(func() {
		if err := s.authorizeNamespace(ctx); err != nil {
			request.err = err
			return
		}
		defaults, ok := s.resourceAccess.Store.(*componentPolicyDefaults)
		if !ok {
			request.access = s.resourceAccess
			return
		}
		parent, ok := defaults.Store.(*namespacepolicy.Store)
		if !ok {
			request.access = s.resourceAccess
			return
		}
		ns, tenant := s.config.NamespaceID, defaults.tenant
		graph, err := s.definitionStore.readCatalogPolicyGraph(ctx, tenant, ns)
		if err != nil {
			request.err = err
			return
		}
		inherited := *parent
		inherited.Store = &discoveryDocuments{Store: parent.Store, graph: graph, tenant: tenant, namespace: ns}
		inherited.Namespace = func(ctx context.Context, id string) (string, error) {
			if _, exists := graph.components[id]; exists {
				return ns, nil
			}
			if parent.Namespace == nil {
				return "", sql.ErrNoRows
			}
			return parent.Namespace(ctx, id)
		}
		scoped := *defaults
		scoped.Store = &inherited
		scoped.hasPolicies = func(ctx context.Context, r authz.Resource) (bool, error) {
			if has, exists := graph.components[r.ID]; exists && r.Kind == "component" && r.Tenant == tenant {
				return has, nil
			}
			return defaults.hasPolicies(ctx, r)
		}
		access := *s.resourceAccess
		access.Store = &scoped
		if provider := access.Provider; provider != nil {
			access.Provider = catalogFacts(sync.OnceValues(func() (authz.Facts, error) { return provider.Resolve(ctx) }))
		}
		request.access = &access
	})
	return request.access, request.err
}
