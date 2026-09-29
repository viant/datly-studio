package host

import (
	"context"
	"errors"
	"maps"
	"strconv"

	access "github.com/viant/authz"
	dexec "github.com/viant/datly/exec"
	"github.com/viant/datly/spec"
)

// catalogAuthorizers retain the same exact source identities as the published
// generation. Skill frontmatter and client arguments never select a policy.
func (s *Service) catalogAuthorizers(components map[spec.Key]string, versions map[string]int, folders map[string]string) (func(context.Context, dexec.ComponentTarget, string) error, func(context.Context, string, string) error, func(context.Context, string) error) {
	components = maps.Clone(components)
	versions = maps.Clone(versions)
	folders = maps.Clone(folders)
	componentResource := func(id string) (access.Resource, error) {
		if id == "" || versions[id] < 1 {
			return access.Resource{}, errors.New("published source identity unavailable")
		}
		return access.Resource{Kind: "component", ID: id, Version: strconv.Itoa(versions[id]), Tenant: s.config.Access.Tenant}, nil
	}
	tool := func(ctx context.Context, target dexec.ComponentTarget, action string) error {
		if err := s.authorizeNamespace(ctx); err != nil {
			return err
		}
		id := components[target.Component]
		if id == "" {
			return errors.New("published component authorization unavailable")
		}
		if s.config.Access == nil {
			return s.authorizeRun(ctx, id)
		}
		resource, err := componentResource(id)
		if err != nil {
			return err
		}
		return s.authorizeResourcePolicy(ctx, resource, action)
	}
	resource := func(ctx context.Context, uri, action string) error {
		if err := s.authorizeNamespace(ctx); err != nil {
			return err
		}
		id := reportForResourceURI(folders, uri)
		if s.config.Access == nil {
			if id == "" {
				return nil
			} // Component-backed resources retain their invocation guard.
			return s.authorizeRun(ctx, id)
		}
		if err := validateResourceURI(uri); err != nil {
			return err
		}
		if len(s.config.Access.ResourceBindings) > 0 {
			return s.authorizeBoundResourceAction(ctx, uri, action)
		}
		owner, err := componentResource(id)
		if err != nil {
			return err
		}
		// Component describe permission includes its authored, immutable files.
		// Explicit skill/resource bindings use their own retrieve policy instead.
		if action == "retrieve" {
			action = "describe"
		}
		return s.authorizeResourcePolicy(ctx, owner, action)
	}
	read := func(ctx context.Context, uri string) error { return resource(ctx, uri, "retrieve") }
	return tool, resource, read
}
