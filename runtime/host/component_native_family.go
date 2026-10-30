package host

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/viant/datly/exec"
	native "github.com/viant/datly/report"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	identity "github.com/viant/forge/backend/resource"
)

// NativeComposition identifies a generated compose target whose original input
// contract has been checked against a source in this exact compiled family.
// It carries no handler or invocation grant.
type NativeComposition struct{ Target, Source exec.ComponentTarget }

// NativeComponentFamily describes only the single selected source's compiler
// output, excluding namespace components and host-injected dependencies. Members
// are stripped metadata. PrepareComposition reuses Datly's bounded native planner
// and returns detached metadata; it can be called only during the callback.
type NativeComponentFamily interface {
	Primary() *registry.RegisteredComponent
	Members() []*registry.RegisteredComponent
	Compositions() []NativeComposition
	PrepareComposition(context.Context, exec.ComponentTarget, native.CompositionDefinition) (*native.PreparedComposition, error)
}

type nativeComponentFamily struct {
	mu           sync.Mutex
	active       bool
	primary      *registry.RegisteredComponent
	members      []*registry.RegisteredComponent
	compositions []NativeComposition
	planners     map[exec.ComponentTarget]rhandler.Handler
	targets      map[exec.ComponentTarget]bool
}

func newNativeComponentFamily(metadata *exactComponentMetadata, ref ComponentReference) (*nativeComponentFamily, error) {
	result := &nativeComponentFamily{active: true, planners: map[exec.ComponentTarget]rhandler.Handler{}, targets: map[exec.ComponentTarget]bool{}}
	byKey := map[spec.Key]*registry.RegisteredComponent{}
	for _, registered := range metadata.family {
		if registered == nil || registered.Component == nil {
			return nil, fmt.Errorf("native family member unavailable")
		}
		key := registered.Component.Key
		if byKey[key] != nil {
			return nil, fmt.Errorf("native family component is ambiguous")
		}
		byKey[key] = registered
		described, err := nativeComponentDescription(registered)
		if err != nil {
			return nil, err
		}
		result.members = append(result.members, described)
		if registered == metadata.registered {
			result.primary = described
		}
		for _, route := range registered.Component.Routes {
			if route == nil {
				return nil, fmt.Errorf("native family route unavailable")
			}
			target := exec.ComponentTarget{Component: key, Route: spec.RouteRef{Method: route.Method, Path: route.Path}}
			// Linked builds have no single-definition compiler provenance. Retain their
			// prior selected-target restriction rather than inferring sibling authority.
			if ref.Kind == "dynamic" || registered == metadata.registered && route.Method == ref.Method && route.Path == ref.Route {
				result.targets[target] = true
			}
		}
	}
	if result.primary == nil {
		return nil, fmt.Errorf("native family primary unavailable")
	}
	for _, registered := range metadata.family {
		source, ok := native.CompositionSource(registered.Handler)
		if !ok {
			continue
		}
		sourceMember := byKey[source.Component]
		if sourceMember == nil || !result.targets[source] || !native.CompositionMatchesSource(registered.Handler, sourceMember) {
			return nil, fmt.Errorf("native composition is not derived from this exact family")
		}
		if len(registered.Component.Routes) != 1 {
			return nil, fmt.Errorf("native composition target is ambiguous")
		}
		route := registered.Component.Routes[0]
		target := exec.ComponentTarget{Component: registered.Component.Key, Route: spec.RouteRef{Method: route.Method, Path: route.Path}}
		if !result.targets[target] {
			continue
		}
		result.compositions = append(result.compositions, NativeComposition{Target: target, Source: source})
		result.planners[target] = registered.Handler
	}
	return result, nil
}
func (f *nativeComponentFamily) Primary() *registry.RegisteredComponent {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.active {
		return nil
	}
	return f.primary
}
func (f *nativeComponentFamily) Members() []*registry.RegisteredComponent {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.active {
		return nil
	}
	return append([]*registry.RegisteredComponent(nil), f.members...)
}
func (f *nativeComponentFamily) Compositions() []NativeComposition {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.active {
		return nil
	}
	return append([]NativeComposition(nil), f.compositions...)
}
func (f *nativeComponentFamily) PrepareComposition(ctx context.Context, target exec.ComponentTarget, definition native.CompositionDefinition) (*native.PreparedComposition, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.active || f.planners[target] == nil {
		return nil, fmt.Errorf("native composition preparation unavailable")
	}
	if ctx == nil {
		return nil, fmt.Errorf("native composition context required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	prepared, err := native.PrepareComposition(f.planners[target], definition)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return prepared, nil
}
func (f *nativeComponentFamily) close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.active = false
	f.primary = nil
	f.members = nil
	f.compositions = nil
	f.planners = nil
	f.targets = nil
}

// The caller supplies only registrations emitted for one authoritative source
// definition, before access contexts or ExactBuildDependencies are appended.
// Sorting the full target/schema inventory gives cold builds a stable identity.
func exactFamilyFingerprint(components []*registry.RegisteredComponent) (string, error) {
	type member struct {
		Component         spec.Key
		Route             spec.RouteRef
		Schema            string
		Metadata          string
		Internal          bool
		CompositionSource *exec.ComponentTarget
	}
	var inventory []member
	byKey := map[spec.Key]*registry.RegisteredComponent{}
	for _, registered := range components {
		if registered == nil || registered.Component == nil || byKey[registered.Component.Key] != nil {
			return "", fmt.Errorf("exact family identity unavailable")
		}
		byKey[registered.Component.Key] = registered
	}
	for _, registered := range components {
		var origin *exec.ComponentTarget
		if source, ok := native.CompositionSource(registered.Handler); ok {
			if !native.CompositionMatchesSource(registered.Handler, byKey[source.Component]) {
				return "", fmt.Errorf("exact composition source contract mismatch")
			}
			origin = &source
		}
		wire, err := json.Marshal(registered.Component)
		if err != nil {
			return "", err
		}
		metadataFingerprint := identity.ContentFingerprint(wire)
		for _, route := range registered.Component.Routes {
			if route == nil {
				return "", fmt.Errorf("exact family route unavailable")
			}
			fingerprint, err := componentSchemaFingerprint(registered, ComponentReference{Method: route.Method, Route: route.Path})
			if err != nil {
				return "", err
			}
			inventory = append(inventory, member{Component: registered.Component.Key, Route: spec.RouteRef{Method: route.Method, Path: route.Path}, Schema: fingerprint, Metadata: metadataFingerprint, Internal: route.Internal, CompositionSource: origin})
		}
	}
	if len(inventory) == 0 {
		return "", fmt.Errorf("exact family inventory unavailable")
	}
	sort.Slice(inventory, func(i, j int) bool {
		a, b := inventory[i], inventory[j]
		if a.Component.String() != b.Component.String() {
			return a.Component.String() < b.Component.String()
		}
		return a.Route.String() < b.Route.String()
	})
	raw, err := json.Marshal(inventory)
	if err != nil {
		return "", err
	}
	return identity.ContentFingerprint(raw), nil
}
