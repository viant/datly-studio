// Package namespacepolicy resolves namespace defaults before Authz evaluates a
// tool's consumption policy. It never grants policy administration by inheritance.
package namespacepolicy

import (
	"context"
	"database/sql"
	"errors"
	"github.com/viant/authz"
)

func Resource(id, tenant string) authz.Resource {
	return authz.Resource{Kind: "namespace", ID: id, Tenant: tenant, Version: "1"}
}

// Store is used for execution, not policy editing. An explicit tool document
// wins in its entirety, including deny and omitted actions. Only a missing
// document can inherit. Errors and malformed policies fail closed.
type Store struct {
	authz.Store
	Namespace func(context.Context, string) (string, error)
}

func (s *Store) Get(ctx context.Context, resource authz.Resource) (authz.Document, error) {
	document, err := s.Store.Get(ctx, resource)
	if !errors.Is(err, sql.ErrNoRows) || resource.Kind != "component" || s.Namespace == nil || ctx.Err() != nil {
		return document, err
	}
	id, err := s.Namespace(ctx, resource.ID)
	if err != nil {
		return authz.Document{}, err
	}
	if id == "" {
		return authz.Document{}, sql.ErrNoRows
	}
	parent := Resource(id, resource.Tenant)
	document, err = s.Store.Get(ctx, parent)
	if err != nil {
		return authz.Document{}, err
	}
	if document.Resource != parent || document.Revision < 1 {
		return authz.Document{}, authz.ErrDenied
	}
	policies := map[string]authz.Policy{}
	for _, action := range []string{"discover", "describe", "execute"} {
		if policy, ok := authz.PolicyForAction(document.Policies, action); ok {
			policies[action] = policy
		}
	}
	return authz.Document{Resource: resource, Revision: document.Revision, Policies: policies}, nil
}
