// Package accesscontext is the server-owned Datly v1 access-context component
// for published Studio components. A published component binds it natively:
//
//	#import('studioaccess','github.com/viant/datly-studio/runtime/accesscontext')
//	#define($_ = $Auth<*studioaccess.Output>(component/GET:/_studio/access/context/<reportId>/project).Required())
//	#define($_ = $ProjectIDs<[]string>(param/Auth.Scope.IDs).Required().WithPredicate(0,'in','t','project_id'))
//
// The runtime host registers one context component per (published component,
// entity dimension) the DQL declares, at that concrete route, so both the
// resource whose decision is served and the dimension the consumer applies to
// its SQL are explicit in the DQL and owned by the server. The handler reads
// only the verified caller credential carried in the invocation context; it has
// no client-bindable input. Its output is the narrowed action decision (local
// policy intersected with any trusted remote decision) for exactly that
// dimension, never the raw facts, so a broader facts list or a decision of
// another dimension cannot replace it. Component-kind and param-kind inputs are
// runtime-protected, so no transport value can supply or override the bound
// context.
package accesscontext

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/viant/datly-studio/sdk/access"
	"github.com/viant/datly/bootstrap"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/handler/custom"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/typecatalog"
	"github.com/viant/x"
	xresponse "github.com/viant/xdatly/response"
)

// ImportPath is the package DQL imports to name Output.
const ImportPath = "github.com/viant/datly-studio/runtime/accesscontext"

// RoutePrefix is followed by the exact published component identifier and the
// entity dimension the consumer applies: /_studio/access/context/{id}/{type}.
const RoutePrefix = "/_studio/access/context/"

// Dependency is one declared (component, entity dimension) context binding.
type Dependency struct {
	ComponentID string
	EntityType  string
}

// Input has no client-bindable fields. The caller credential is the verified
// bearer the runtime host already attached to the invocation context.
type Input struct{}

// Context is the verified principal context for one narrowed execute decision.
// AllowedEntities uses the canonical allowedEntities shape (entity type -> IDs)
// but carries only the declared dimension, holding exactly the narrowed
// decision: consuming AllowedEntities[entityType] can never bypass the
// local/remote intersection that produced Scope.
type Context struct {
	Subject         string              `json:"subject"`
	Tenant          string              `json:"tenant"`
	Roles           []string            `json:"roles"`
	Exposures       []string            `json:"exposures"`
	AllowedEntities map[string][]string `json:"allowedEntities"`
	ValidUntil      time.Time           `json:"validUntil"`
}

// Scope is the entity-bounded execute decision for the component this context
// serves. IDs never exceed the intersected decision.
type Scope struct {
	EntityType string   `json:"entityType"`
	IDs        []string `json:"ids"`
}

// Output is what a consuming component binds.
type Output struct {
	Context *Context `json:"context"`
	Scope   *Scope   `json:"scope"`
}

// Decider yields the verified facts and the narrowed execute decision for the
// component a context instance serves.
type Decider func(context.Context) (access.Facts, access.Decision, error)

// Route is the concrete component route serving one dependency.
func (d Dependency) Route() string { return RoutePrefix + d.ComponentID + "/" + d.EntityType }

// Reference is the DQL component dependency reference for one dependency.
func (d Dependency) Reference() string { return "GET:" + d.Route() }

// Valid accepts identifiers usable as exactly one path segment each.
func (d Dependency) Valid() bool { return validSegment(d.ComponentID) && validSegment(d.EntityType) }

func validSegment(value string) bool {
	if value == "" || strings.TrimSpace(value) != value || strings.ContainsAny(value, "/\\?#%") || value == "." || value == ".." {
		return false
	}
	for _, r := range value {
		if r <= ' ' || r == 0x7f {
			return false
		}
	}
	return true
}

// Component describes the context component for one dependency.
func Component(dependency Dependency) (*spec.Component, error) {
	if !dependency.Valid() {
		return nil, fmt.Errorf("access context %q/%q is invalid", dependency.ComponentID, dependency.EntityType)
	}
	return &spec.Component{
		Key:    spec.Key{Kind: spec.KindComponent, Scope: ImportPath, Name: "Context_" + dependency.ComponentID + "_" + dependency.EntityType},
		Name:   "AccessContext",
		Routes: []*spec.Route{{Method: "GET", Path: dependency.Route()}},
	}, nil
}

// Handler adapts a Decider to the typed component handler for one dimension.
func Handler(entityType string, decide Decider) rhandler.TypedHandler {
	return custom.NewFunc[Input, Output](func(ctx context.Context, _ *Input) (*Output, error) {
		if decide == nil {
			return nil, forbidden("access context decider is unavailable")
		}
		facts, decision, err := decide(ctx)
		if err != nil {
			return nil, forbidden("access context denied")
		}
		return Convert(facts, decision, entityType)
	})
}

// Register builds the registered context component for one dependency.
func Register(dependency Dependency, decide Decider) (*registry.RegisteredComponent, error) {
	component, err := Component(dependency)
	if err != nil {
		return nil, err
	}
	artifact, err := bootstrap.BuildArtifact(bootstrap.ArtifactInput{Component: component, InputType: reflect.TypeOf(Input{}), OutputType: reflect.TypeOf(Output{})})
	if err != nil {
		return nil, fmt.Errorf("access context %s: %w", dependency.Route(), err)
	}
	return &registry.RegisteredComponent{Component: artifact.Component, Input: artifact.Input, OutputType: reflect.TypeOf(Output{}), Handler: Handler(dependency.EntityType, decide)}, nil
}

// RegisterTypes makes Output resolvable to DQL through ImportPath.
func RegisterTypes(catalog *typecatalog.Catalog) error {
	if catalog == nil {
		return errors.New("type catalog is required")
	}
	for _, typ := range []reflect.Type{reflect.TypeOf(Output{}), reflect.TypeOf(Context{}), reflect.TypeOf(Scope{})} {
		if err := catalog.Register(typecatalog.TypeOriginPackage, x.NewType(typ, x.WithPkgPath(ImportPath))); err != nil {
			return err
		}
	}
	return nil
}

// DependsOn reports every distinct access context the component binds, in
// declaration order. It fails on a reference below RoutePrefix that is not an
// exact GET:/_studio/access/context/{component}/{entityType} route.
func DependsOn(component *spec.Component) ([]Dependency, error) {
	if component == nil {
		return nil, nil
	}
	var result []Dependency
	seen := map[Dependency]bool{}
	for _, parameter := range spec.EffectiveParameters(component.Parameters) {
		if parameter == nil || !strings.EqualFold(strings.TrimSpace(parameter.Source.Kind), string(spec.KindComponent)) {
			continue
		}
		reference := strings.TrimSpace(parameter.Source.Name)
		method, path, ok := strings.Cut(reference, ":")
		if !ok || !strings.HasPrefix(path, RoutePrefix) {
			continue
		}
		if !strings.EqualFold(method, "GET") {
			return nil, fmt.Errorf("access context dependency %q must use GET", reference)
		}
		componentID, entityType, ok := strings.Cut(strings.TrimPrefix(path, RoutePrefix), "/")
		dependency := Dependency{ComponentID: componentID, EntityType: entityType}
		if !ok || !dependency.Valid() {
			return nil, fmt.Errorf("access context dependency %q must name exactly one component and one entity dimension", reference)
		}
		if seen[dependency] {
			continue
		}
		seen[dependency] = true
		result = append(result, dependency)
	}
	return result, nil
}

// Convert maps verified facts and a narrowed decision to the bound output for
// one declared dimension. It fails closed: unverified or expired facts, an
// unbounded decision, an empty or mixed entity set, or a decision of another
// dimension all deny.
func Convert(facts access.Facts, decision access.Decision, entityType string) (*Output, error) {
	if strings.TrimSpace(facts.Subject) == "" || strings.TrimSpace(facts.Tenant) == "" {
		return nil, forbidden("access context requires a verified principal")
	}
	if !facts.ValidUntil.After(time.Now()) {
		return nil, forbidden("access context is expired")
	}
	if _, err := facts.FlatEntities(); err != nil {
		return nil, forbidden("access context entities are malformed")
	}
	if !decision.Bounded {
		return nil, forbidden("access context requires an entity-bounded execute decision")
	}
	if len(decision.Entities) == 0 {
		return nil, forbidden("access context decision grants no entities")
	}
	if !validSegment(entityType) {
		return nil, forbidden("access context dimension is invalid")
	}
	scope := &Scope{EntityType: entityType}
	seen := map[string]bool{}
	for _, entity := range decision.Entities {
		if entity.Type != entityType {
			return nil, forbidden("access context decision is not for the declared entity dimension")
		}
		if entity.ID == "" || seen[entity.ID] {
			return nil, forbidden("access context decision is malformed")
		}
		seen[entity.ID] = true
		scope.IDs = append(scope.IDs, entity.ID)
	}
	roles := append([]string(nil), facts.Roles...)
	exposures := append([]string(nil), facts.Exposures...)
	sort.Strings(roles)
	sort.Strings(exposures)
	// The canonical map exposes only the narrowed decision, never the broader
	// facts of the same dimension and no dimension this decision did not cover.
	allowed := map[string][]string{entityType: append([]string(nil), scope.IDs...)}
	return &Output{
		Context: &Context{Subject: facts.Subject, Tenant: facts.Tenant, Roles: roles, Exposures: exposures, AllowedEntities: allowed, ValidUntil: facts.ValidUntil},
		Scope:   scope,
	}, nil
}

func forbidden(message string) error {
	return &xresponse.Error{Code: 403, Cause: errors.New(message)}
}
