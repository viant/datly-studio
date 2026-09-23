package host

import (
	"context"
	"github.com/viant/scy/auth/jwt"
	"net/http"
	"strings"
)

type providerIdentitiesKey struct{}

func matchesProvider(claims *jwt.Claims, provider OAuthProvider) bool {
	if claims == nil || strings.TrimSpace(claims.Subject) == "" || claims.Issuer != provider.Issuer {
		return false
	}
	return claims.VerifyAudience(provider.Audience, true)
}

func hasScopes(claims *jwt.Claims, required []string) bool {
	scopes := map[string]bool{}
	for _, scope := range strings.Fields(claims.Scope) {
		scopes[scope] = true
	}
	for _, scope := range required {
		if !scopes[scope] {
			return false
		}
	}
	return true
}

// Authenticate credentials once; the component hook enforces its exact provider
// and scopes for HTTP, MCP tool calls and resource reads alike.
func (s *Service) componentAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		identities := map[string]*jwt.Claims{}
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			for name, service := range s.providerVerifiers {
				claims, err := service.VerifyClaims(ctx, parts[1])
				if err == nil && matchesProvider(claims, s.config.Authentication.Providers[name]) {
					identities[name] = claims
				}
			}
			if s.verifier != nil {
				if claims, err := s.verifier.VerifyClaims(ctx, parts[1]); err == nil {
					ctx = context.WithValue(ctx, verifiedClaimsKey{}, claims)
				}
			}
		}
		legacy, _ := ctx.Value(verifiedClaimsKey{}).(*jwt.Claims)
		public := strings.EqualFold(s.config.Authentication.DefaultMode, "public")
		for _, policy := range s.config.Authentication.Components {
			public = public || policy.Public
		}
		if !public && len(identities) == 0 && legacy == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(ctx, providerIdentitiesKey{}, identities)))
	})
}
