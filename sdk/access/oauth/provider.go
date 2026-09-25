// Package oauth resolves ACL facts from a dedicated, trusted OAuth/OIDC issuer.
package oauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
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
	Tenant    string   `json:"tenant"`
	Roles     []string `json:"roles"`
	Exposures []string `json:"exposures"`
	// EntityGroups is the canonical typed allowedEntities claim.
	EntityGroups access.EntityGroups `json:"-"`
	// AllowedEntities accepts legacy flat Go callers. Tokens are serialized in
	// grouped form; received flat tokens are validated and normalized.
	AllowedEntities []access.Entity `json:"-"`
}

type wireClaims struct {
	jwt.RegisteredClaims
	Tenant          string          `json:"tenant"`
	Roles           []string        `json:"roles,omitempty"`
	Exposures       []string        `json:"exposures,omitempty"`
	AllowedEntities json.RawMessage `json:"allowedEntities,omitempty"`
}

func (c Claims) MarshalJSON() ([]byte, error) {
	if c.EntityGroups != nil && c.AllowedEntities != nil {
		if _, err := (access.Facts{EntityGroups: c.EntityGroups, Entities: c.AllowedEntities}).FlatEntities(); err != nil {
			return nil, err
		}
	}
	groups := c.EntityGroups
	if groups == nil && c.AllowedEntities != nil {
		groups = groupLegacyForWire(c.AllowedEntities)
	}
	var raw json.RawMessage
	if groups != nil {
		var err error
		raw, err = json.Marshal(groups)
		if err != nil {
			return nil, err
		}
	}
	return json.Marshal(wireClaims{RegisteredClaims: c.RegisteredClaims, Tenant: c.Tenant, Roles: c.Roles, Exposures: c.Exposures, AllowedEntities: raw})
}

// Preserve malformed legacy values on the wire so verification can deny them
// at the trust boundary, instead of silently dropping them during signing.
func groupLegacyForWire(flat []access.Entity) access.EntityGroups {
	groups := access.EntityGroups{}
	for _, entity := range flat {
		groups[entity.Type] = append(groups[entity.Type], access.EntityID(entity.ID))
	}
	return groups
}

func (c *Claims) UnmarshalJSON(raw []byte) error {
	// encoding/json otherwise accepts duplicate claim keys with the last value
	// winning. Duplicate authority claims are ambiguous and always denied.
	claimDecoder := json.NewDecoder(bytes.NewReader(raw))
	opening, err := claimDecoder.Token()
	if err != nil || opening != json.Delim('{') {
		return access.ErrDenied
	}
	seenAuthority := false
	for claimDecoder.More() {
		keyToken, err := claimDecoder.Token()
		if err != nil {
			return access.ErrDenied
		}
		key, ok := keyToken.(string)
		if !ok {
			return access.ErrDenied
		}
		if strings.EqualFold(key, "allowedEntities") {
			if seenAuthority || key != "allowedEntities" {
				return access.ErrDenied
			}
			seenAuthority = true
		}
		var value json.RawMessage
		if err := claimDecoder.Decode(&value); err != nil {
			return access.ErrDenied
		}
	}
	closing, err := claimDecoder.Token()
	if err != nil || closing != json.Delim('}') {
		return access.ErrDenied
	}
	if _, err := claimDecoder.Token(); err != io.EOF {
		return access.ErrDenied
	}
	var wire wireClaims
	if err := json.Unmarshal(raw, &wire); err != nil {
		return err
	}
	*c = Claims{RegisteredClaims: wire.RegisteredClaims, Tenant: wire.Tenant, Roles: wire.Roles, Exposures: wire.Exposures}
	if len(wire.AllowedEntities) == 0 {
		return nil
	}
	var groups access.EntityGroups
	switch wire.AllowedEntities[0] {
	case '{':
		var err error
		groups, err = access.DecodeEntityGroups(wire.AllowedEntities)
		if err != nil {
			return err
		}
	case '[':
		var entries []json.RawMessage
		if err := json.Unmarshal(wire.AllowedEntities, &entries); err != nil || entries == nil {
			return access.ErrDenied
		}
		flat := make([]access.Entity, 0, len(entries))
		for _, entry := range entries {
			dec := json.NewDecoder(bytes.NewReader(entry))
			opening, err := dec.Token()
			if err != nil || opening != json.Delim('{') {
				return access.ErrDenied
			}
			seen := map[string]bool{}
			for dec.More() {
				keyToken, err := dec.Token()
				if err != nil {
					return access.ErrDenied
				}
				key, ok := keyToken.(string)
				if !ok || (key != "type" && key != "id") || seen[key] {
					return access.ErrDenied
				}
				seen[key] = true
				var value json.RawMessage
				if err := dec.Decode(&value); err != nil {
					return access.ErrDenied
				}
			}
			closing, err := dec.Token()
			if err != nil || closing != json.Delim('}') || !seen["type"] || !seen["id"] {
				return access.ErrDenied
			}
			if _, err := dec.Token(); err != io.EOF {
				return access.ErrDenied
			}
			var entity access.Entity
			if err := json.Unmarshal(entry, &entity); err != nil {
				return access.ErrDenied
			}
			flat = append(flat, entity)
		}
		var err error
		groups, err = access.GroupEntities(flat)
		if err != nil {
			return err
		}
	default:
		return access.ErrDenied
	}
	flat, err := access.NormalizeEntityGroups(groups)
	if err != nil {
		return err
	}
	c.EntityGroups, c.AllowedEntities = groups, flat
	return nil
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
	flat, err := access.NormalizeEntityGroups(claims.EntityGroups)
	if err != nil {
		return access.Facts{}, access.ErrDenied
	}
	for _, values := range [][]string{claims.Roles, claims.Exposures} {
		for _, v := range values {
			if v == "" {
				return access.Facts{}, access.ErrDenied
			}
		}
	}
	return access.Facts{Subject: claims.Subject, Tenant: claims.Tenant, Issuer: claims.Issuer, Roles: claims.Roles, Exposures: claims.Exposures, EntityGroups: claims.EntityGroups, Entities: flat, ValidUntil: claims.ExpiresAt.Time}, nil
}
