package artifactrelease

import (
	"context"
	"errors"

	"github.com/viant/authz"
	"github.com/viant/forge/backend/artifactcohort"
	contract "github.com/viant/forge/backend/artifactrelease"
)

// AudienceEvaluator uses the same Authz engine and verified Facts provider as
// ordinary producer policy checks. It examines only the derived cohort join
// document present in the current published manifest; a named policy ID is
// never interpreted as a grant without its compiled active projection.
func AudienceEvaluator(publisher *artifactcohort.Publisher, provider authz.Provider, decisions authz.DecisionProvider) func(context.Context, contract.Channel, authz.NamedReference) (bool, error) {
	return func(ctx context.Context, channel contract.Channel, reference authz.NamedReference) (bool, error) {
		if ctx == nil || ctx.Err() != nil || publisher == nil || provider == nil {
			return false, contract.ErrInvalid
		}
		name := contract.ReleaseCohortName(channel.Key, reference)
		policies, store, err := publisher.CohortPolicies(name)
		if err != nil {
			return false, err
		}
		var resource authz.Resource
		found := false
		for _, policy := range policies {
			if policy.Kind == "cohort" && policy.ID == name && policy.Version == "logical" {
				if found {
					return false, contract.ErrInvalid
				}
				resource = policy
				found = true
			}
		}
		if !found {
			return false, nil
		}
		service := &authz.Service{Store: store, Provider: provider, Decisions: decisions}
		decision, _, revision, err := service.AuthorizeWithStatus(ctx, authz.Request{Resource: resource, Action: artifactcohort.PolicyAction})
		if errors.Is(err, authz.ErrDenied) || errors.Is(err, authz.ErrIdentityDenied) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if revision != int64(channel.Revision) || decision.Bounded {
			return false, contract.ErrBoundedMembership
		}
		return true, nil
	}
}
