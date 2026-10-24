package host

import (
	"context"
	"database/sql"
	"errors"

	"github.com/viant/authz"
	sdkrelease "github.com/viant/datly-studio/sdk/artifactrelease"
	"github.com/viant/forge/backend/artifactcohort"
	windowprotocol "github.com/viant/forge/backend/window/contract"
)

type releaseComponentContextKey struct{}
type releaseComponentLease struct {
	Root                  artifactcohort.RootRef
	ManifestHash          string
	ComponentID, Revision string
}

func (s *Service) releaseController() *sdkrelease.Controller {
	if s == nil {
		return nil
	}
	controller, _ := s.config.ReleaseManagement.(*sdkrelease.Controller)
	return controller
}

// defaultCreatorUsePolicy identifies only the plain ownership membership
// restriction that a named release may replace. Explicit deny, composite
// audience rules, required scopes and entity bounds remain mandatory.
func (s *Service) defaultCreatorUsePolicy(ctx context.Context, resource authz.Resource) error {
	if s == nil || s.studio == nil || s.resourceAccess == nil || s.resourceAccess.Store == nil {
		return authz.ErrDenied
	}
	var owner string
	if err := s.studio.QueryRowContext(ctx, `SELECT owner_id FROM components WHERE id=? AND status='active' AND deleted_at IS NULL`, resource.ID).Scan(&owner); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return authz.ErrDenied
		}
		return err
	}
	doc, err := s.resourceAccess.Store.Get(ctx, resource)
	if err != nil {
		return err
	}
	policy, found := authz.PolicyForAction(doc.Policies, "execute")
	if !found || policy.Mode != "protected" || policy.Rule == nil || policy.Rule.Kind != "subject" || policy.Rule.Value != owner || len(policy.Rule.Rules) > 0 || policy.Rule.Entity != nil || len(policy.RequiredScopes) > 0 || policy.EntityType != "" {
		return authz.ErrDenied
	}
	return nil
}

func (s *Service) admitReleasedComponent(ctx context.Context, ref ComponentReference, pin windowprotocol.ComponentBinding) (releaseComponentLease, error) {
	controller := s.releaseController()
	if controller == nil || s.config.Access == nil {
		return releaseComponentLease{}, authz.ErrDenied
	}
	resource := authz.Resource{Tenant: s.config.Access.Tenant, Kind: "component", ID: ref.ID, Version: ref.Revision}
	if err := s.defaultCreatorUsePolicy(ctx, resource); err != nil {
		return releaseComponentLease{}, err
	}
	snapshot, err := controller.ActiveSnapshot(ctx)
	if err != nil {
		return releaseComponentLease{}, err
	}
	for _, selection := range snapshot.Selections {
		for _, binding := range selection.Bindings {
			if binding.ComponentKind != ref.Kind || binding.ComponentID != ref.ID || binding.ComponentRevision != ref.Revision || binding.Method != ref.Method || binding.Route != ref.Route || binding.SchemaFingerprint != pin.SchemaFingerprint || binding.ContentFingerprint != pin.ContentFingerprint {
				continue
			}
			hash, grantErr := controller.AdmitRoot(ctx, binding.Root)
			if grantErr != nil {
				continue
			}
			if hash != snapshot.ManifestHash {
				return releaseComponentLease{}, authz.ErrDenied
			}
			return releaseComponentLease{Root: binding.Root, ManifestHash: hash, ComponentID: ref.ID, Revision: ref.Revision}, nil
		}
	}
	return releaseComponentLease{}, authz.ErrDenied
}

func (s *Service) fenceReleasedComponent(ctx context.Context, lease releaseComponentLease) error {
	controller := s.releaseController()
	if controller == nil {
		return authz.ErrDenied
	}
	resource := authz.Resource{Tenant: s.config.Access.Tenant, Kind: "component", ID: lease.ComponentID, Version: lease.Revision}
	if err := s.defaultCreatorUsePolicy(ctx, resource); err != nil {
		return err
	}
	return controller.FenceRoot(ctx, lease.Root, lease.ManifestHash)
}

func admittedReleaseFor(ctx context.Context, resource authz.Resource) bool {
	lease, ok := ctx.Value(releaseComponentContextKey{}).(releaseComponentLease)
	return ok && lease.ComponentID == resource.ID && lease.Revision == resource.Version && lease.ManifestHash != ""
}
