package host

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/viant/datly/application"
	"github.com/viant/datly/runtime"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	identity "github.com/viant/forge/backend/resource"
)

// ExactBuildDependencyResolver is an opt-in embedding-host capability. It runs
// after exact-source authorization and compilation, before runtime initialization,
// on every isolated dynamic build, including metadata-only release verification.
// It must return fresh registrations and a fingerprint of all executable support
// content/configuration. It cannot change the reader, its authorization or any
// publication. Returning nil leaves the stock exact runtime unchanged.
type ExactBuildDependencyResolver func(context.Context, string, ComponentReference) (*ExactBuildDependencies, error)

// ExactBuildDependencies owns internal native component dependencies for one
// exact reader generation. Components must have only internal routes and no MCP
// exposures. Shutdown releases generation resources on success or failure.
// The fingerprint participates in the exact dispatch pin, never the reader's
// compiled-contract cache; the resolver runs even on a compiled-contract hit.
type ExactBuildDependencies struct {
	Components         []*registry.RegisteredComponent
	RuntimeOptions     []runtime.Option
	ContentFingerprint string
	Shutdown           func(context.Context) error
}

func (s *Service) addExactBuildDependencies(ctx context.Context, ref ComponentReference, build *application.Build) (_ string, err error) {
	if s.config.ExactBuildDependencyResolver == nil {
		return "", nil
	}
	dependencies, err := s.config.ExactBuildDependencyResolver(ctx, s.config.NamespaceID, ref)
	transferred := false
	defer func() {
		if transferred {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if dependencies != nil && dependencies.Shutdown != nil {
			err = errors.Join(err, dependencies.Shutdown(cleanupCtx))
		}
		if build.Shutdown != nil {
			err = errors.Join(err, build.Shutdown(cleanupCtx))
		}
	}()
	if err != nil {
		return "", fmt.Errorf("resolve exact build dependencies: %w", err)
	}
	if dependencies == nil {
		transferred = true
		return "", nil
	}
	if !(identity.ResourceCandidate{Kind: identity.WorkingCandidate, ContentFingerprint: dependencies.ContentFingerprint}).Valid() {
		return "", fmt.Errorf("exact dependency fingerprint is required")
	}
	keys := map[spec.Key]bool{}
	routes := map[spec.RouteRef]bool{}
	for _, component := range build.Components {
		keys[component.Component.Key] = true
		for _, route := range component.Component.Routes {
			if route != nil {
				routes[spec.RouteRef{Method: route.Method, Path: route.Path}] = true
			}
		}
	}
	for _, registered := range dependencies.Components {
		if registered == nil || registered.Component == nil {
			return "", fmt.Errorf("exact dependency component is unavailable")
		}
		component := registered.Component
		if keys[component.Key] || len(component.Routes) == 0 {
			return "", fmt.Errorf("exact dependency conflicts with component")
		}
		keys[component.Key] = true
		for _, route := range component.Routes {
			if route == nil || !route.Internal || len(route.MCP) != 0 {
				return "", fmt.Errorf("exact dependency routes must be internal without MCP exposure")
			}
			key := spec.RouteRef{Method: route.Method, Path: route.Path}
			if routes[key] {
				return "", fmt.Errorf("exact dependency route conflicts with existing route")
			}
			routes[key] = true
		}
	}
	build.Components = append(build.Components, dependencies.Components...)
	build.RuntimeOptions = append(build.RuntimeOptions, dependencies.RuntimeOptions...)
	shutdown := build.Shutdown
	build.Shutdown = func(ctx context.Context) error {
		var result error
		if dependencies.Shutdown != nil {
			result = dependencies.Shutdown(ctx)
		}
		if shutdown != nil {
			result = errors.Join(result, shutdown(ctx))
		}
		return result
	}
	transferred = true
	return dependencies.ContentFingerprint, nil
}
