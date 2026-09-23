package host

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/viant/scy/auth/jwt"
)

// Provider endpoints give clients an unambiguous authorization server choice.
// Their canonical resource URLs are configured, never derived from Host headers.
func (s *Service) oauthDiscovery(next http.Handler) http.Handler {
	base := strings.TrimRight(s.config.Authentication.PublicMCPURL, "/")
	if base == "" {
		return s.authenticated(next)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const prefix = "/.well-known/oauth-protected-resource/"
		metadata := strings.HasPrefix(r.URL.Path, prefix)
		endpoint := r.URL.Path
		if metadata {
			endpoint = "/" + strings.TrimPrefix(endpoint, prefix)
		}
		parts := strings.Split(strings.Trim(endpoint, "/"), "/")
		if len(parts) != 3 || parts[0] != "oauth" || parts[2] != "mcp" {
			s.authenticated(next).ServeHTTP(w, r)
			return
		}
		name := parts[1]
		provider, ok := s.config.Authentication.Providers[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		resource := base + "/oauth/" + url.PathEscape(name) + "/mcp"
		discovery := base + prefix + "oauth/" + url.PathEscape(name) + "/mcp"
		if metadata {
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"resource": resource, "authorization_servers": []string{provider.Issuer}, "bearer_methods_supported": []string{"header"}})
			return
		}
		parts = strings.Fields(r.Header.Get("Authorization"))
		var claims *jwt.Claims
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			if verifier := s.providerVerifiers[name]; verifier != nil {
				value, err := verifier.VerifyClaims(r.Context(), parts[1])
				if err == nil && matchesProvider(value, provider) {
					claims = value
				}
			}
		}
		if claims == nil {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf("Bearer resource_metadata=%q", discovery))
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), providerIdentitiesKey{}, map[string]*jwt.Claims{name: claims})
		request := r.Clone(ctx)
		copied := *r.URL
		copied.Path = "/mcp"
		copied.RawPath = ""
		request.URL = &copied
		next.ServeHTTP(w, request)
	})
}
