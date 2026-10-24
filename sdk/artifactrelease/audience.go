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
		manifest, err := publisher.Snapshot()
		if err != nil {
			return false, err
		}
		name := contract.ReleaseCohortName(channel.Key, reference)
		var resource authz.Resource
		found := false
		for _, cohort := range manifest.Cohorts {
			if cohort.Name == name && cohort.PolicyResource.Kind == "cohort" && cohort.PolicyResource.ID == name && cohort.PolicyResource.Version == "logical" && cohort.Tenant == cohort.PolicyResource.Tenant {
				if found {
					return false, contract.ErrInvalid
				}
				resource = cohort.PolicyResource
				found = true
			}
		}
		if !found {
			return false, nil
		}
		store, err := authz.NewStaticStore(manifest.Policies)
		if err != nil {
			return false, err
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
