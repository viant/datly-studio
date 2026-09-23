package host

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/viant/datly-studio/sdk/access"
	"github.com/viant/datly-studio/sdk/access/oauth"
	accessstore "github.com/viant/datly-studio/store/sql/access"
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

// authorizeScopedComponent authorizes execution of one published component and,
// for an entity-bounded decision, binds the caller's authorized IDs into the
// component's deployment-declared typed scope input. HTTP and MCP share this
// path. It fails closed: a bounded decision without a binding, an unbounded
// decision on a scoped component, an entity dimension other than the bound
// one, or an ID that does not convert canonically all deny before execution.
func (s *Service) authorizeScopedComponent(ctx context.Context, r access.Resource, binding *resolvedScopeBinding) error {
	if s.resourceAccess == nil {
		return &xresponse.Error{Code: http.StatusForbidden, Cause: access.ErrDenied}
	}
	d, err := s.resourceAccess.Authorize(ctx, access.Request{Resource: r, Action: "execute"})
	if err != nil {
		return &xresponse.Error{Code: http.StatusForbidden, Cause: access.ErrDenied}
	}
	if !d.Bounded {
		if binding != nil {
			return &xresponse.Error{Code: http.StatusForbidden, Cause: errors.New("scoped component requires an entity-bounded policy decision")}
		}
		return nil
	}
	if binding == nil {
		return &xresponse.Error{Code: http.StatusForbidden, Cause: errors.New("typed runtime scope binding is required")}
	}
	if err = binding.bind(ctx, d); err != nil {
		return &xresponse.Error{Code: http.StatusForbidden, Cause: errors.New("authorized scope cannot be bound to the component input")}
	}
	return nil
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
