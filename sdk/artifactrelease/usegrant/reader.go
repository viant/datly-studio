// Package usegrant checks committed release component bindings from Studio's
// shared database. It is safe for the separate Studio API process: no runtime
// Publisher memory, browser-selected scope, or permanent component ACL row is
// treated as release authority.
package usegrant

import (
	"context"
	"errors"
	"reflect"

	"github.com/viant/authz"
	"github.com/viant/datly-studio/sdk/artifactrelease/sqlstore"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
)

type Reader struct {
	Store             *sqlstore.Store
	Scope             sqlstore.Scope
	VerifyScope       func(context.Context) (sqlstore.Scope, error)
	Provider          authz.Provider
	Decisions         authz.DecisionProvider
	NamedDocuments    func(context.Context, []authz.NamedReference) ([]authz.Document, error)
	FenceDependencies func(context.Context, []authz.NamedDependency) error
	// NativeConstraints retains explicit component deny, account/entity scope,
	// and row predicates. A producer may return deny for unsupported bounds.
	NativeConstraints func(context.Context, sqlstore.ComponentBinding, string) error
}

func (r *Reader) Allowed(ctx context.Context, componentID, revision, permission string) (bool, error) {
	if r == nil || r.Store == nil || r.VerifyScope == nil || r.Provider == nil || r.NamedDocuments == nil || r.FenceDependencies == nil || r.NativeConstraints == nil || ctx == nil || ctx.Err() != nil || componentID == "" || (permission != "view" && permission != "run") || permission == "run" && revision == "" {
		return false, contract.ErrInvalid
	}
	actual, err := r.VerifyScope(ctx)
	if err != nil {
		return false, err
	}
	if actual != r.Scope {
		return false, contract.ErrInvalid
	}
	channels, err := r.Store.ListChannels(ctx)
	if err != nil {
		return false, err
	}
	for _, channel := range channels {
		if channel.ActiveBundleHash == "" {
			continue
		}
		bundle, readErr := r.Store.Bundle(ctx, channel.ActiveBundleHash)
		if readErr != nil {
			return false, readErr
		}
		bindings, readErr := r.Store.BundleBindings(ctx, bundle)
		if readErr != nil {
			return false, readErr
		}
		matched := false
		for _, binding := range bindings {
			if binding.ComponentID != componentID || revision != "" && binding.ComponentRevision != revision {
				continue
			}
			if readErr = r.NativeConstraints(ctx, binding, permission); readErr != nil {
				if errors.Is(readErr, authz.ErrDenied) {
					continue
				}
				return false, readErr
			}
			matched = true
			break
		}
		if !matched {
			continue
		}
		persisted, readErr := r.Store.ActiveDependencies(ctx, channel.Key)
		if readErr != nil || len(persisted) == 0 {
			return false, contract.ErrInvalid
		}
		if readErr = r.FenceDependencies(ctx, persisted); readErr != nil {
			return false, readErr
		}
		docs, readErr := r.NamedDocuments(ctx, channel.Policies)
		if readErr != nil {
			return false, readErr
		}
		compiled, readErr := contract.CompileCohorts(ctx, channel, bundle, docs)
		if readErr != nil {
			return false, readErr
		}
		if !reflect.DeepEqual(compiled.Dependencies, persisted) {
			return false, contract.ErrConflict
		}
		store, readErr := authz.NewStaticStore(compiled.Policies)
		if readErr != nil {
			return false, readErr
		}
		service := &authz.Service{Store: store, Provider: r.Provider, Decisions: r.Decisions}
		for _, cohort := range compiled.Cohorts {
			decision, _, observed, decisionErr := service.AuthorizeWithStatus(ctx, authz.Request{Resource: cohort.PolicyResource, Action: artifactcohort.PolicyAction})
			if errors.Is(decisionErr, authz.ErrDenied) || errors.Is(decisionErr, authz.ErrIdentityDenied) {
				continue
			}
			if decisionErr != nil {
				return false, decisionErr
			}
			if observed != int64(channel.Revision) || decision.Bounded {
				return false, contract.ErrBoundedMembership
			}
			if readErr = r.FenceDependencies(ctx, persisted); readErr != nil {
				return false, readErr
			}
			// Policy heads can remain unchanged while a channel moves to another
			// bundle. Fence the exact committed selection as well as its policy.
			current, readErr := r.Store.Channel(ctx, channel.Key)
			if readErr != nil {
				return false, readErr
			}
			if !reflect.DeepEqual(current, channel) {
				return false, contract.ErrConflict
			}
			return true, nil
		}
	}
	return false, nil
}
