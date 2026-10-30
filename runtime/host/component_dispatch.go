package host

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/viant/authz"
	studiors "github.com/viant/datly-studio/runtime/resources"
	studiohost "github.com/viant/datly-studio/studio/host"
	"github.com/viant/datly/application"
	dexec "github.com/viant/datly/exec"
	mcptool "github.com/viant/datly/mcp/tool"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	"github.com/viant/datly/typecatalog"
	identity "github.com/viant/forge/backend/resource"
	windowprotocol "github.com/viant/forge/backend/window/contract"
)

// ComponentReference is a deployment-owned dispatch mapping. An incoming input
// cannot supply a route, select another component or choose an active alias.
type ComponentReference struct {
	Kind     string `json:"kind" yaml:"kind"`
	ID       string `json:"id" yaml:"id"`
	Revision string `json:"revision" yaml:"revision"`
	Method   string `json:"method" yaml:"method"`
	Route    string `json:"route" yaml:"route"`
}

// LinkedComponentSource binds an application build to an explicit binary/source
// artifact revision. Build returns fresh native Datly components; their native
// authorization, SQL predicates and typed input/output contracts remain active.
type LinkedComponentSource struct {
	ID, Revision, ArtifactFingerprint string
	Build                             func(context.Context, *typecatalog.Catalog) (*application.Build, error)
}

func validateComponentReference(ref ComponentReference) error {
	if ref.ID == "" || strings.TrimSpace(ref.ID) != ref.ID || ref.Revision == "" || strings.TrimSpace(ref.Revision) != ref.Revision || ref.Revision == "active" || ref.Revision == "latest" || ref.Revision == "working" {
		return fmt.Errorf("component dispatch requires an exact revision")
	}
	if ref.Kind != "dynamic" && ref.Kind != "linked" {
		return fmt.Errorf("unsupported component source")
	}
	if ref.Method != http.MethodGet && ref.Method != http.MethodPost || !strings.HasPrefix(ref.Route, "/") || strings.ContainsAny(ref.Route, "?#") || strings.HasPrefix(ref.Route, "//") {
		return fmt.Errorf("invalid component route")
	}
	if ref.Kind == "dynamic" {
		version, err := strconv.Atoi(ref.Revision)
		if err != nil || version < 1 || strconv.Itoa(version) != ref.Revision {
			return fmt.Errorf("dynamic component revision must be a canonical positive version")
		}
	}
	return nil
}

// ResolveComponentBinding describes the exact executable source. The caller
// must authorize the returned pin before dispatch; no revision is inferred.
func (s *Service) ResolveComponentBinding(ctx context.Context, ref ComponentReference) (windowprotocol.ComponentBinding, error) {
	manager, binding, err := s.componentRuntime(ctx, ref)
	if err != nil {
		return windowprotocol.ComponentBinding{}, err
	}
	defer manager.Shutdown(context.Background())
	return binding, nil
}

// ExecuteComponentJSON recompiles the same selected revision, checks content
// and schema identity, executes native Datly authorization, then refuses source
// drift before releasing data. It never uses the current publication manager.
func (s *Service) ExecuteComponentJSON(ctx context.Context, ref ComponentReference, pinned windowprotocol.ComponentBinding, inputs map[string]any) (json.RawMessage, error) {
	output, err := s.executeExactComponent(ctx, ref, pinned, func(ctx context.Context, manager *application.Manager, _ *exactComponentMetadata) (any, error) {
		return executeRuntimeJSON(ctx, s.authenticated(manager), ref.Method, ref.Route, inputs)
	}, nil)
	if err != nil {
		return nil, err
	}
	return output.(json.RawMessage), nil
}

// executeExactComponent is the common source, policy and identity fence for all
// exact execution transports. The execution result remains private until every
// completion check has succeeded.
func (s *Service) executeExactComponent(ctx context.Context, ref ComponentReference, pinned windowprotocol.ComponentBinding, execute func(context.Context, *application.Manager, *exactComponentMetadata) (any, error), verify func(windowprotocol.ComponentBinding, *exactComponentMetadata) error) (any, error) {
	metadata := &exactComponentMetadata{}
	manager, current, err := s.componentRuntimeInspected(ctx, ref, true, metadata)
	metadataDenied := false
	if err != nil && errors.Is(err, authz.ErrDenied) && s.releaseController() != nil {
		manager, current, err = s.componentRuntimeInspected(ctx, ref, false, metadata)
		metadataDenied = true
	}
	if err != nil {
		return nil, err
	}
	defer manager.Shutdown(context.Background())
	if err := windowprotocol.ValidateComponentDispatch(&pinned, current); err != nil {
		return nil, err
	}
	if pinned.Kind == "" || pinned.ContentFingerprint == "" {
		return nil, fmt.Errorf("complete component pin is required")
	}
	if verify != nil {
		if err := verify(current, metadata); err != nil {
			return nil, err
		}
	}

	var observedPolicyRevision int64
	var releaseLease *releaseComponentLease
	if s.config.Access != nil && s.resourceAccess != nil {
		policyService := s.resourceAccess
		if ref.Kind == "dynamic" {
			version, _ := strconv.Atoi(ref.Revision)
			policyService = s.publishedAuthorizer(map[string]int{ref.ID: version}).resourceAccess
		}
		_, _, observedPolicyRevision, err = policyService.AuthorizeWithStatus(ctx, authz.Request{Resource: authz.Resource{Kind: "component", ID: ref.ID, Version: ref.Revision, Tenant: s.config.Access.Tenant}, Action: "execute"})
		if err != nil || metadataDenied {
			if err != nil && !errors.Is(err, authz.ErrDenied) {
				return nil, err
			}
			lease, admissionErr := s.admitReleasedComponent(ctx, ref, pinned)
			if admissionErr != nil {
				return nil, admissionErr
			}
			releaseLease = &lease
			ctx = context.WithValue(ctx, releaseComponentContextKey{}, lease)
		}
	}
	var observed authz.Facts
	var observedAuthority []byte
	if s.resourceAccess != nil && s.resourceAccess.Provider != nil {
		observed, err = s.resourceAccess.Provider.Resolve(ctx)
		if err != nil || !observed.ValidUntil.After(time.Now()) {
			return nil, authz.ErrIdentityDenied
		}
		snapshot := observed
		snapshot.ValidUntil = time.Time{}
		observedAuthority, err = json.Marshal(snapshot)
		if err != nil {
			return nil, err
		}
	}
	if !observed.ValidUntil.IsZero() {
		bounded, cancel := context.WithDeadline(ctx, observed.ValidUntil)
		defer cancel()
		ctx = bounded
	}
	output, err := execute(ctx, manager, metadata)
	if err != nil {
		return nil, err
	}
	// Re-resolve exactly the same revision after execution. This also detects
	// changed SQL/resources/linked source contracts during a long reader call.
	var fresh windowprotocol.ComponentBinding
	if verify != nil {
		// Native family proof is checked on the existing completion rebuild. JSON
		// dispatch keeps its primary-only observation path and legacy pin identity.
		metadata := &exactComponentMetadata{}
		verification, observed, readErr := s.componentRuntimeInspected(ctx, ref, releaseLease == nil, metadata)
		fresh, err = observed, readErr
		if err == nil {
			err = verify(fresh, metadata)
		}
		if verification != nil {
			err = errors.Join(err, shutdownNativeMetadata(ctx, verification.Shutdown))
		}
	} else if releaseLease != nil {
		verification, observed, readErr := s.componentRuntimeWithAuthority(ctx, ref, false)
		if verification != nil {
			_ = verification.Shutdown(context.Background())
		}
		fresh, err = observed, readErr
	} else {
		fresh, err = s.ResolveComponentBinding(ctx, ref)
	}
	if err != nil {
		return nil, err
	}
	if err := windowprotocol.ValidateComponentDispatch(&pinned, fresh); err != nil {
		return nil, err
	}
	if releaseLease != nil {
		if err = s.fenceReleasedComponent(ctx, *releaseLease); err != nil {
			return nil, err
		}
	} else if s.config.Access != nil && s.resourceAccess != nil {
		policyService := s.resourceAccess
		if ref.Kind == "dynamic" {
			version, _ := strconv.Atoi(ref.Revision)
			policyService = s.publishedAuthorizer(map[string]int{ref.ID: version}).resourceAccess
		}
		_, _, currentRevision, currentErr := policyService.AuthorizeWithStatus(ctx, authz.Request{Resource: authz.Resource{Kind: "component", ID: ref.ID, Version: ref.Revision, Tenant: s.config.Access.Tenant}, Action: "execute"})
		if currentErr != nil || currentRevision != observedPolicyRevision {
			return nil, authz.ErrDenied
		}
	}
	if s.resourceAccess != nil && s.resourceAccess.Provider != nil {
		currentFacts, currentErr := s.resourceAccess.Provider.Resolve(ctx)
		if currentErr != nil || !observed.ValidUntil.After(time.Now()) || !currentFacts.ValidUntil.After(time.Now()) {
			return nil, authz.ErrIdentityDenied
		}
		currentFacts.ValidUntil = time.Time{}
		currentAuthority, marshalErr := json.Marshal(currentFacts)
		if marshalErr != nil || !bytes.Equal(observedAuthority, currentAuthority) {
			return nil, authz.ErrIdentityDenied
		}
	}
	return output, nil
}

func (s *Service) componentRuntime(ctx context.Context, ref ComponentReference) (_ *application.Manager, _ windowprotocol.ComponentBinding, err error) {
	return s.componentRuntimeWithAuthority(ctx, ref, true)
}

// Metadata-only construction is private to trusted startup. No request handler
// can select it, and the resulting manager is closed before caching any bytes.
func (s *Service) componentRuntimeWithAuthority(ctx context.Context, ref ComponentReference, authorize bool) (_ *application.Manager, _ windowprotocol.ComponentBinding, err error) {
	return s.componentRuntimeInspected(ctx, ref, authorize, nil)
}

func (s *Service) componentRuntimeInspected(ctx context.Context, ref ComponentReference, authorize bool, metadata *exactComponentMetadata) (_ *application.Manager, _ windowprotocol.ComponentBinding, err error) {
	if s == nil || ctx == nil || ctx.Err() != nil || s.studio == nil {
		return nil, windowprotocol.ComponentBinding{}, fmt.Errorf("component runtime is unavailable")
	}
	if err := validateComponentReference(ref); err != nil {
		return nil, windowprotocol.ComponentBinding{}, err
	}
	types, err := (studiohost.Config{PredicatePackages: s.config.PredicatePackages, CodecPackages: s.config.CodecPackages}).RuntimeTypes()
	if err != nil {
		return nil, windowprotocol.ComponentBinding{}, err
	}
	manager, err := application.New(types)
	if err != nil {
		return nil, windowprotocol.ComponentBinding{}, err
	}
	succeeded := false
	defer func() {
		if !succeeded {
			_ = manager.Shutdown(context.Background())
		}
	}()
	binding := windowprotocol.ComponentBinding{ID: ref.ID, Revision: ref.Revision, Kind: ref.Kind}
	if ref.Kind == "dynamic" {
		version, _ := strconv.Atoi(ref.Revision)
		sourceDefinition, err := s.exactDefinition(ctx, ref.ID, version)
		if err != nil {
			return nil, binding, err
		}
		authorizer := s.publishedAuthorizer(map[string]int{ref.ID: version})
		if authorize {
			if err := authorizer.authorizeNamespace(ctx); err != nil {
				return nil, binding, err
			}
			if s.config.Access != nil {
				if err := authorizer.authorizeResourcePolicy(ctx, authz.Resource{Kind: "component", ID: ref.ID, Version: ref.Revision, Tenant: s.config.Access.Tenant}, "describe"); err != nil {
					return nil, binding, err
				}
			} else if err := authorizer.authorizeRun(ctx, ref.ID); err != nil {
				return nil, binding, err
			}
		}
		err = manager.Reload(ctx, application.Request{Revision: 1, Compile: func(ctx context.Context, seed *typecatalog.Catalog) (*application.Build, error) {
			captured := false
			// This callback observes only the native compiler's complete output
			// for the single selected definition. Access contexts and hook
			// dependencies are appended later and never become family members.
			var family []*registry.RegisteredComponent
			build, err := s.compileDefinitions(ctx, seed, []definition{sourceDefinition}, func(source definition, registered *registry.RegisteredComponent, resources *studiors.Loaded) error {
				if source.reportID != ref.ID || source.versionNo != version {
					return fmt.Errorf("exact family source identity changed")
				}
				family = append(family, registered)
				if !hasComponentRoute(registered, ref) {
					return nil
				}
				if captured {
					return fmt.Errorf("component route is ambiguous")
				}
				captured = true
				if metadata != nil {
					metadata.registered = registered
				}
				fingerprint, hashErr := componentSchemaFingerprint(registered, ref)
				if hashErr != nil {
					return hashErr
				}
				binding.SchemaFingerprint = fingerprint
				raw, err := json.Marshal(struct {
					ID, Scope, Name, Connector, Driver, DSN, Secret, DQL, Resources string
					Version                                                         int
				}{source.reportID, source.scope, source.name, source.connector, source.driver, source.dsn, source.secretRef, source.dql, resources.Fingerprints[studiors.Version{ReportID: source.reportID, VersionNo: source.versionNo}], source.versionNo})
				if err != nil {
					return err
				}
				binding.ContentFingerprint = identity.ContentFingerprint(raw)
				return nil
			})
			if err != nil {
				return nil, err
			}
			if !captured {
				if build.Shutdown != nil {
					_ = build.Shutdown(ctx)
				}
				return nil, fmt.Errorf("selected component does not declare the mapped route")
			}
			if metadata != nil {
				metadata.family = family
			}
			dependencyFingerprint, dependencyErr := s.addExactBuildDependencies(ctx, ref, build)
			if dependencyErr != nil {
				return nil, dependencyErr
			}
			if dependencyFingerprint != "" {
				binding.ContentFingerprint = identity.ContentFingerprint([]byte(binding.ContentFingerprint + ":" + dependencyFingerprint))
			}
			if metadata != nil {
				metadata.authorize = build.HTTP.Authorize
			}
			return build, nil
		}})
		if err != nil {
			return nil, binding, err
		}
	} else {
		if authorize {
			if err := s.authorizeNamespace(ctx); err != nil {
				return nil, binding, err
			}
			if s.config.Access == nil || s.resourceAccess == nil {
				return nil, binding, authz.ErrDenied
			}
			if err := s.authorizeResourcePolicy(ctx, authz.Resource{Kind: "component", ID: ref.ID, Version: ref.Revision, Tenant: s.config.Access.Tenant}, "describe"); err != nil {
				return nil, binding, err
			}
		}
		var source *LinkedComponentSource
		for i := range s.config.LinkedComponents {
			candidate := &s.config.LinkedComponents[i]
			if candidate.ID == ref.ID && candidate.Revision == ref.Revision {
				if source != nil {
					return nil, binding, fmt.Errorf("linked component revision is ambiguous")
				}
				source = candidate
			}
		}
		if source == nil || source.Build == nil || !(identity.ResourceCandidate{Kind: identity.WorkingCandidate, ContentFingerprint: source.ArtifactFingerprint}).Valid() {
			return nil, binding, fmt.Errorf("linked component revision is unavailable")
		}
		err = manager.Reload(ctx, application.Request{Revision: 1, Compile: func(ctx context.Context, seed *typecatalog.Catalog) (*application.Build, error) {
			build, err := source.Build(ctx, seed)
			if err != nil {
				return nil, err
			}
			if build == nil {
				return nil, fmt.Errorf("linked component build is unavailable")
			}
			transferred := false
			defer func() {
				if !transferred && build.Shutdown != nil {
					_ = build.Shutdown(context.Background())
				}
			}()
			found := false
			for _, registered := range build.Components {
				if registered == nil || registered.Component == nil {
					continue
				}
				if err := requireReaderOnlyContract(registered.Component); err != nil {
					return nil, err
				}
				if !hasComponentRoute(registered, ref) {
					continue
				}
				if found {
					return nil, fmt.Errorf("linked component route is ambiguous")
				}
				found = true
				if metadata != nil {
					metadata.registered = registered
				}
				binding.SchemaFingerprint, err = componentSchemaFingerprint(registered, ref)
				if err != nil {
					return nil, err
				}
				raw, err := json.Marshal(struct {
					Artifact  string
					Component *spec.Component
				}{source.ArtifactFingerprint, registered.Component})
				if err != nil {
					return nil, err
				}
				binding.ContentFingerprint = identity.ContentFingerprint(raw)
			}
			if !found {
				return nil, fmt.Errorf("linked component route is unavailable")
			}
			componentIDs := map[spec.Key]string{}
			for _, registered := range build.Components {
				if registered != nil && registered.Component != nil {
					componentIDs[registered.Component.Key] = ref.ID
				}
			}
			contexts, binds, contextErr := s.accessContextsForVersions(build.Components, componentIDs, map[string]string{ref.ID: ref.Revision})
			if contextErr != nil {

				return nil, contextErr
			}
			build.Components = append(build.Components, contexts...)
			previous := build.HTTP.Authorize
			build.HTTP.Authorize = func(ctx context.Context, r *http.Request, target dexec.ComponentTarget) error {
				if err := s.authorizeNamespace(ctx); err != nil {
					return err
				}
				if s.config.Access == nil || s.resourceAccess == nil {
					return authz.ErrDenied
				}
				resource := authz.Resource{Kind: "component", ID: ref.ID, Version: ref.Revision, Tenant: s.config.Access.Tenant}
				if err := s.authorizeComponentExecution(ctx, resource, binds[ref.ID]); err != nil {
					if !admittedReleaseFor(ctx, resource) || !errors.Is(err, authz.ErrDenied) {
						return err
					}
				}
				if previous != nil {
					return previous(ctx, r, target)
				}
				return nil
			}
			if metadata != nil {
				metadata.family = []*registry.RegisteredComponent{metadata.registered}
				metadata.authorize = build.HTTP.Authorize
			}
			transferred = true
			return build, nil
		}})
		if err != nil {
			return nil, binding, err
		}
	}
	succeeded = true
	return manager, binding, nil
}
func hasComponentRoute(registered *registry.RegisteredComponent, ref ComponentReference) bool {
	if registered == nil || registered.Component == nil {
		return false
	}
	for _, route := range registered.Component.Routes {
		if route != nil && route.Method == ref.Method && route.Path == ref.Route {
			return true
		}
	}
	return false
}
func componentSchemaFingerprint(registered *registry.RegisteredComponent, ref ComponentReference) (string, error) {
	if registered.Input == nil {
		return "", fmt.Errorf("component input contract is unavailable")
	}
	contract, ok := registered.Input.ForRoute(spec.RouteRef{Method: ref.Method, Path: ref.Route})
	if !ok {
		return "", fmt.Errorf("component route contract is unavailable")
	}
	plan, err := mcptool.NewCompiler().Compile(mcptool.Input{Component: registered.Component.Key, Exposure: &spec.MCPExposure{Kind: spec.MCPExposureTool, Name: "component.dispatch"}, Contract: contract, OutputType: registered.OutputType, Output: registered.Output, TransportReady: registered.Output != nil && registered.Output.TransportReady()})
	if err != nil {
		return "", err
	}
	metadata := plan.Metadata()
	raw, err := json.Marshal([]any{metadata.InputSchema, metadata.OutputSchema})
	if err != nil {
		return "", err
	}
	return identity.ContentFingerprint(raw), nil
}
