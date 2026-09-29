package host

import (
	"context"
	"database/sql"
	"errors"
	"maps"
	"strconv"

	"github.com/viant/authz"
)

// componentPolicyDefaults supplies read-only public actions for exact component
// versions in this published artifact. It never defaults unknown identities,
// policy administration, malformed policies, or database failures.
type componentPolicyDefaults struct {
	authz.Store
	tenant      string
	versions    map[string]int
	hasPolicies func(context.Context, authz.Resource) (bool, error)
}

func (s *componentPolicyDefaults) Get(ctx context.Context, resource authz.Resource) (authz.Document, error) {
	document, err := s.Store.Get(ctx, resource)
	if !errors.Is(err, sql.ErrNoRows) || ctx.Err() != nil {
		return document, err
	}
	if resource.Kind != "component" || resource.Tenant != s.tenant || s.versions[resource.ID] < 1 || resource.Version != strconv.Itoa(s.versions[resource.ID]) {
		return document, err
	}
	if s.hasPolicies == nil {
		return document, err
	}
	existing, presenceErr := s.hasPolicies(ctx, resource)
	if presenceErr != nil {
		return authz.Document{}, presenceErr
	}
	if existing {
		return document, err
	}
	return authz.Document{Resource: resource, Revision: 1, Policies: map[string]authz.Policy{
		"discover": {Mode: "public"}, "describe": {Mode: "public"}, "execute": {Mode: "public"},
	}}, nil
}

// publishedAuthorizer pins defaults to one immutable artifact's source mapping.
// Namespace and tenant verification remain independent authorization gates.
func (s *Service) publishedAuthorizer(versions map[string]int) *Service {
	if s.resourceAccess == nil || s.resourceAccess.Store == nil || s.config.Access == nil {
		return s
	}
	access := *s.resourceAccess
	access.Store = &componentPolicyDefaults{Store: access.Store, tenant: s.config.Access.Tenant, versions: maps.Clone(versions), hasPolicies: s.definitionStore.HasPolicy}
	return &Service{config: s.config, studio: s.studio, runAccessStore: s.runAccessStore, resourceAccess: &access, definitionStore: s.definitionStore}
}
