package accessconfig

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/viant/authz"
	"github.com/viant/authz/gating"
	"github.com/viant/authz/oauth"
)

// ClaimsVerifier authenticates with the configured keys before exposing claims.
// Issuer/audience restrictions are optional; expiry, signature and subject are not.
type ClaimsVerifier struct {
	key              jwtlib.Keyfunc
	issuer, audience string
}

func NewClaimsVerifier(config Config) (*ClaimsVerifier, error) {
	key, err := configuredKeyfunc(config.JWTValidator, config.PublicKeyFile, config.CertURL)
	if err != nil {
		return nil, err
	}
	return &ClaimsVerifier{key: key, issuer: config.Issuer, audience: config.Audience}, nil
}
func (v *ClaimsVerifier) Verify(ctx context.Context) (jwtlib.MapClaims, error) {
	if v == nil || ctx == nil || ctx.Err() != nil {
		return nil, authz.ErrDenied
	}
	options := []jwtlib.ParserOption{jwtlib.WithValidMethods([]string{"RS256"}), jwtlib.WithExpirationRequired(), jwtlib.WithIssuedAt(), jwtlib.WithJSONNumber()}
	if v.issuer != "" {
		options = append(options, jwtlib.WithIssuer(v.issuer))
	}
	if v.audience != "" {
		options = append(options, jwtlib.WithAudience(v.audience))
	}
	claims := jwtlib.MapClaims{}
	token, err := jwtlib.ParseWithClaims(oauth.Bearer(ctx), claims, v.key, options...)
	if err != nil || token == nil || !token.Valid {
		return nil, authz.ErrDenied
	}
	subject, err := claims.GetSubject()
	if err != nil || strings.TrimSpace(subject) == "" || strings.TrimSpace(subject) != subject {
		return nil, authz.ErrDenied
	}
	issuer, err := claims.GetIssuer()
	if err != nil || strings.TrimSpace(issuer) == "" {
		return nil, authz.ErrDenied
	}
	return claims, nil
}

type verifiedIdentity struct {
	verifier *ClaimsVerifier
	tenant   string
}

func newVerifiedIdentity(config Config) (*verifiedIdentity, error) {
	v, err := NewClaimsVerifier(config)
	if err != nil {
		return nil, err
	}
	return &verifiedIdentity{verifier: v, tenant: config.Tenant}, nil
}
func (p *verifiedIdentity) Resolve(ctx context.Context) (authz.Facts, error) {
	principal, err := p.ResolvePrincipal(ctx)
	return principal.Facts, err
}
func (p *verifiedIdentity) ResolvePrincipal(ctx context.Context) (gating.Principal, error) {
	if ctx == nil || ctx.Err() != nil {
		return gating.Principal{}, authz.ErrDenied
	}
	if scope, _ := ctx.Value(identityScopeKey{}).(*identityScope); scope != nil {
		return scope.resolve(ctx, p)
	}
	return p.readPrincipal(ctx)
}
func (p *verifiedIdentity) readPrincipal(ctx context.Context) (gating.Principal, error) {
	claims, err := p.verifier.Verify(ctx)
	if err != nil {
		return gating.Principal{}, err
	}
	subject, _ := claims.GetSubject()
	issuer, _ := claims.GetIssuer()
	expiry, _ := claims.GetExpirationTime()
	tenant := p.tenant
	if tenant == "" {
		tenant, _ = claims["tenant"].(string)
	}
	facts := authz.Facts{Subject: subject, Issuer: issuer, Tenant: tenant, ValidUntil: expiry.Time}
	facts.Roles = claimStrings(claims["roles"])
	facts.Exposures = claimStrings(claims["exposures"])
	account := claimString(claims["account_id"])
	if camel := claimString(claims["accountId"]); camel != "" {
		if account != "" && account != camel {
			return gating.Principal{}, authz.ErrDenied
		}
		account = camel
	}
	groups := claimStrings(claims["membershipGroups"])
	if len(groups) == 0 {
		groups = claimStrings(claims["membership_groups"])
	}
	raw, _ := json.Marshal([]any{issuer, subject, tenant, account, facts.Roles, facts.Exposures, groups})
	revision := fmt.Sprintf("%x", sha256.Sum256(raw))
	facts.AuthorityRevision = revision
	return gating.Principal{Facts: facts, AccountID: account, MembershipGroups: groups, IdentityRevision: revision}, nil
}
func (p *verifiedIdentity) Account(ctx context.Context, facts authz.Facts) (string, error) {
	principal, err := p.ResolvePrincipal(ctx)
	if err != nil || principal.Facts.Subject != facts.Subject || principal.Facts.Issuer != facts.Issuer || principal.Facts.Tenant != facts.Tenant || principal.AccountID == "" {
		return "", authz.ErrDenied
	}
	return principal.AccountID, nil
}
func (p *verifiedIdentity) AuthorityRevision(ctx context.Context, facts authz.Facts, account string) (string, time.Time, error) {
	principal, err := p.ResolvePrincipal(ctx)
	if err != nil || principal.Facts.Subject != facts.Subject || principal.Facts.Issuer != facts.Issuer || principal.Facts.Tenant != facts.Tenant || principal.AccountID != account {
		return "", time.Time{}, authz.ErrDenied
	}
	return principal.IdentityRevision, principal.Facts.ValidUntil, nil
}
func claimString(value any) string {
	switch value := value.(type) {
	case string:
		return value
	case json.Number:
		return value.String()
	}
	return ""
}
func claimStrings(value any) []string {
	result := []string{}
	switch value := value.(type) {
	case []any:
		for _, item := range value {
			if text, ok := item.(string); ok {
				result = append(result, text)
			}
		}
	case []string:
		result = append(result, value...)
	}
	return result
}
