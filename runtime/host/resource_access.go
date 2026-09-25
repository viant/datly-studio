package host

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/viant/datly-studio/runtime/accesscontext"
	"github.com/viant/datly-studio/sdk/access"
	"github.com/viant/datly-studio/sdk/access/oauth"
	accessstore "github.com/viant/datly-studio/store/sql/access"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	xresponse "github.com/viant/xdatly/response"
)

func (s *Service) initResourceAccess() error {
	c := s.config.Access
	if c == nil {
		return nil
	}
	data, err := os.ReadFile(c.PublicKeyFile)
	if err != nil {
		return err
	}
	key, err := jwtlib.ParseRSAPublicKeyFromPEM(data)
	if err != nil {
		return err
	}
	provider, err := oauth.New(oauth.Config{Issuer: c.Issuer, Audience: c.Audience, Algorithms: []string{"RS256"}, Keyfunc: func(*jwtlib.Token) (any, error) { return key, nil }})
	if err != nil {
		return err
	}
	s.resourceAccess = &access.Service{Store: &accessstore.Store{DB: s.studio}, Provider: provider, Decisions: s.config.DecisionProvider}
	return nil
}

func (s *Service) authorizeResourcePolicy(ctx context.Context, r access.Resource, action string) error {
	if s.resourceAccess == nil {
		return &xresponse.Error{Code: http.StatusForbidden, Cause: access.ErrDenied}
	}
	d, err := s.resourceAccess.Authorize(ctx, access.Request{Resource: r, Action: action})
	if err != nil {
		return &xresponse.Error{Code: http.StatusForbidden, Cause: access.ErrDenied}
	}
	if d.Bounded {
		return &xresponse.Error{Code: http.StatusForbidden, Cause: errors.New("typed runtime scope binding is required")}
	}
	return nil
}

// authorizeComponentExecution authorizes execution of one published component.
// HTTP and MCP share this path. An entity-bounded decision is enforceable only
// when the component binds its own access-context component natively (see
// runtime/accesscontext); a bounded decision on a component that does not is
// denied before execution rather than released unscoped. The bound context
// component then re-evaluates the same narrowed decision when the consuming
// component binds its input, so the scope that reaches SQL is never broader.
func (s *Service) authorizeComponentExecution(ctx context.Context, r access.Resource, bindsAccessContext bool) error {
	if s.resourceAccess == nil {
		return &xresponse.Error{Code: http.StatusForbidden, Cause: access.ErrDenied}
	}
	d, err := s.resourceAccess.Authorize(ctx, access.Request{Resource: r, Action: "execute"})
	if err != nil {
		return &xresponse.Error{Code: http.StatusForbidden, Cause: access.ErrDenied}
	}
	if d.Bounded && !bindsAccessContext {
		return &xresponse.Error{Code: http.StatusForbidden, Cause: errors.New("entity-bounded policy requires the component to bind its access context")}
	}
	return nil
}

// accessContexts registers one server-owned access-context component per
// published component that declares the dependency, at that component's
// concrete route. A component may bind only its own context, and binding one
// requires generic resource access to be configured.
func (s *Service) accessContexts(registrations []*registry.RegisteredComponent, reportByComponent map[spec.Key]string, versionByReport map[string]int) ([]*registry.RegisteredComponent, map[string]bool, error) {
	binds := map[string]bool{}
	seen := map[accesscontext.Dependency]bool{}
	var contexts []*registry.RegisteredComponent
	for _, registered := range registrations {
		if registered == nil || registered.Component == nil {
			continue
		}
		reportID := reportByComponent[registered.Component.Key]
		dependencies, err := accesscontext.DependsOn(registered.Component)
		if err != nil {
			return nil, nil, fmt.Errorf("report %s: %w", reportID, err)
		}
		for _, dependency := range dependencies {
			if dependency.ComponentID != reportID {
				return nil, nil, fmt.Errorf("report %s binds the access context of %q; a component may bind only its own", reportID, dependency.ComponentID)
			}
			if s.config.Access == nil || s.resourceAccess == nil {
				return nil, nil, fmt.Errorf("report %s binds an access context, which requires generic resource access configuration", reportID)
			}
			if seen[dependency] {
				continue
			}
			seen[dependency] = true
			binds[reportID] = true
			resource := access.Resource{Kind: "component", ID: reportID, Version: strconv.Itoa(versionByReport[reportID]), Tenant: s.config.Access.Tenant}
			service := s.resourceAccess
			registration, err := accesscontext.Register(dependency, func(ctx context.Context) (access.Facts, access.Decision, error) {
				facts, err := service.Provider.Resolve(ctx)
				if err != nil {
					return access.Facts{}, access.Decision{}, err
				}
				decision, err := service.Authorize(ctx, access.Request{Resource: resource, Action: "execute"})
				if err != nil {
					return access.Facts{}, access.Decision{}, err
				}
				return facts, decision, nil
			})
			if err != nil {
				return nil, nil, err
			}
			contexts = append(contexts, registration)
		}
	}
	return contexts, binds, nil
}

func (s *Service) authorizeBoundResource(ctx context.Context, uri string) error {
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme == "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil || strings.Contains(parsed.Path, "\\") {
		return &xresponse.Error{Code: http.StatusForbidden, Cause: access.ErrDenied}
	}
	for _, segment := range strings.Split(parsed.Path, "/") {
		if segment == "." || segment == ".." {
			return &xresponse.Error{Code: http.StatusForbidden, Cause: access.ErrDenied}
		}
	}
	var selected access.Resource
	length := 0
	for prefix, r := range s.config.Access.ResourceBindings {
		if strings.HasPrefix(uri, prefix) && len(prefix) > length {
			selected = r
			length = len(prefix)
		}
	}
	if length == 0 {
		return &xresponse.Error{Code: http.StatusForbidden, Cause: access.ErrDenied}
	}
	return s.authorizeResourcePolicy(ctx, selected, "retrieve")
}

func (s *Service) resourceCredential(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		values := r.Header.Values("Authorization")
		if len(values) > 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if len(values) == 1 {
			parts := strings.Fields(values[0])
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx = oauth.WithBearer(ctx, parts[1])
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
