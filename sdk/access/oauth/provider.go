// Package oauth resolves ACL facts from a dedicated, trusted OAuth/OIDC issuer.
package oauth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/viant/datly-studio/sdk"
	"github.com/viant/datly-studio/sdk/access"
)

// Config separates the authorization issuer/audience from authoring or model
// credentials. Keyfunc must use deployment-owned keys/JWKS, never a token URL.
type Config struct {
	Issuer     string
	Audience   string
	Algorithms []string
	Keyfunc    jwt.Keyfunc
}

type Claims struct {
	jwt.RegisteredClaims
	Tenant          string          `json:"tenant"`
	Roles           []string        `json:"roles"`
	Exposures       []string        `json:"exposures"`
	AllowedEntities []access.Entity `json:"allowedEntities"`
}

type Provider struct{ config Config }
type tokenKey struct{}

// WithBearer carries the incoming credential to Resolve; it does not mark the
// token verified. Resolve validates its signature and registered claims.
func WithBearer(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenKey{}, token)
}

func New(config Config) (*Provider, error) {
	if config.Issuer == "" || config.Audience == "" || config.Keyfunc == nil || len(config.Algorithms) == 0 {
		return nil, errors.New("OAuth issuer, audience, keys and algorithms required")
	}
	for _, a := range config.Algorithms {
		switch a {
		case "RS256", "RS384", "RS512", "ES256", "ES384", "ES512", "EdDSA":
		default:
			return nil, errors.New("OAuth signing algorithm must be asymmetric")
		}
	}
	config.Algorithms = append([]string(nil), config.Algorithms...)
	return &Provider{config: config}, nil
}

var _ access.Provider = (*Provider)(nil)

func (p *Provider) Resolve(ctx context.Context) (access.Facts, error) {
	if err := ctx.Err(); err != nil {
		return access.Facts{}, err
	}
	bearer, _ := ctx.Value(tokenKey{}).(string)
	if bearer == "" {
		if credential, ok := sdk.VerifiedCredentialFromContext(ctx); ok {
			bearer = credential.Bearer
		}
	}
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(bearer, claims, p.config.Keyfunc, jwt.WithValidMethods(p.config.Algorithms), jwt.WithIssuer(p.config.Issuer), jwt.WithAudience(p.config.Audience), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || claims.Subject == "" || claims.Tenant == "" || claims.Tenant == "*" || claims.ExpiresAt == nil || !claims.ExpiresAt.After(time.Now()) {
		return access.Facts{}, access.ErrDenied
	}
	for _, e := range claims.AllowedEntities {
		if e.Type == "" || e.ID == "" {
			return access.Facts{}, access.ErrDenied
		}
	}
	for _, values := range [][]string{claims.Roles, claims.Exposures} {
		for _, v := range values {
			if v == "" {
				return access.Facts{}, access.ErrDenied
			}
		}
	}
	return access.Facts{Subject: claims.Subject, Tenant: claims.Tenant, Issuer: claims.Issuer, Roles: claims.Roles, Exposures: claims.Exposures, Entities: claims.AllowedEntities, ValidUntil: claims.ExpiresAt.Time}, nil
}
