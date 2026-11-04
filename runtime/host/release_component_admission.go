package host

import (
	"context"
	"database/sql"
	"errors"

	"github.com/viant/authz"
	sdkrelease "github.com/viant/datly-studio/sdk/artifactrelease"
	releasevalue "github.com/viant/datly-studio/sdk/artifactrelease/value"
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
	if s == nil || s.reportCapabilities == nil || s.resourceAccess == nil || s.resourceAccess.Store == nil {
		return authz.ErrDenied
	}
	capability, err := s.reportCapabilities.Read(ctx, resource.ID, "")
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return authz.ErrDenied
		}
		return err
	}
	if capability.ComponentStatus != "active" {
		return authz.ErrDenied
	}
	doc, err := s.resourceAccess.Store.Get(ctx, resource)
	if err != nil {
		return err
	}
	policy, found := authz.PolicyForAction(doc.Policies, "execute")
	if !found || !plainCreatorPolicy(policy, capability.OwnerId) {
		return authz.ErrDenied
	}
	return nil
}

func plainCreatorPolicy(policy authz.Policy, owner string) bool {
	return policy.Mode == "protected" && policy.Rule != nil && policy.Rule.Kind == "subject" && policy.Rule.Value == owner && len(policy.Rule.Rules) == 0 && policy.Rule.Entity == nil && len(policy.RequiredScopes) == 0 && policy.EntityType == ""
}

// A producer-admitted release already joins describe and execute membership.
// Missing source metadata actions add no Studio authoring requirement; present
// restrictions may be delegated only when they are the same plain creator rule.
func (s *Service) defaultCreatorDescriptionPolicy(ctx context.Context, resource authz.Resource) error {
	capability, err := s.reportCapabilities.Read(ctx, resource.ID, "")
	if err != nil {
		return err
	}
	if capability.ComponentStatus != "active" {
		return authz.ErrDenied
	}
	doc, err := s.resourceAccess.Store.Get(ctx, resource)
	if err != nil {
		return err
	}
	policy, found := authz.PolicyForAction(doc.Policies, "describe")
	if found && !plainCreatorPolicy(policy, capability.OwnerId) {
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
	roots, manifest, err := controller.ComponentRoots(ctx, releasevalue.ComponentBinding{ComponentKind: ref.Kind, ComponentID: ref.ID, ComponentRevision: ref.Revision, Method: ref.Method, Route: ref.Route, SchemaFingerprint: pin.SchemaFingerprint, ContentFingerprint: pin.ContentFingerprint})
	if err != nil {
		return releaseComponentLease{}, err
	}
	for _, root := range roots {
		hash, grantErr := controller.AdmitRoot(ctx, root)
		if grantErr != nil {
			continue
		}
		if hash != manifest {
			return releaseComponentLease{}, authz.ErrDenied
		}
		return releaseComponentLease{Root: root, ManifestHash: hash, ComponentID: ref.ID, Revision: ref.Revision}, nil
	}
	return releaseComponentLease{}, authz.ErrDenied
}

func (s *Service) authorizeReleasedComponentDescription(ctx context.Context, ref ComponentReference, pin windowprotocol.ComponentBinding) error {
	lease, err := s.admitReleasedComponent(ctx, ref, pin)
	if err != nil {
		return err
	}
	resource := authz.Resource{Kind: "component", ID: ref.ID, Version: ref.Revision, Tenant: s.config.Access.Tenant}
	admitted := context.WithValue(ctx, releaseComponentContextKey{}, lease)
	if err = s.authorizeNamespaceResource(admitted, resource); err != nil {
		return err
	}
	if err = s.authorizeResourcePolicy(ctx, resource, "describe"); err != nil {
		if !errors.Is(err, authz.ErrDenied) {
			return err
		}
		if err = s.defaultCreatorDescriptionPolicy(ctx, resource); err != nil {
			return err
		}
	}
	return s.fenceReleasedComponent(admitted, lease)
}

func (s *Service) fenceReleasedComponent(ctx context.Context, lease releaseComponentLease) error {
	controller := s.releaseController()
	if controller == nil {
		return authz.ErrDenied
	}
	resource := authz.Resource{Tenant: s.config.Access.Tenant, Kind: "component", ID: lease.ComponentID, Version: lease.Revision}
	if err := s.authorizeNamespaceResource(ctx, resource); err != nil {
		return err
	}
	if err := s.defaultCreatorUsePolicy(ctx, resource); err != nil {
		return err
	}
	return controller.FenceRoot(ctx, lease.Root, lease.ManifestHash)
}

func admittedReleaseFor(ctx context.Context, resource authz.Resource) bool {
	lease, ok := ctx.Value(releaseComponentContextKey{}).(releaseComponentLease)
	return ok && lease.ComponentID == resource.ID && lease.Revision == resource.Version && lease.ManifestHash != ""
}
