package studioapi

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/viant/scy/auth/jwt"
)

type tokenClaimsVerifier interface {
	VerifyClaims(context.Context, string) (*jwt.Claims, error)
}

// Signature validation stays with the configured SCY verifier. Expiry and
// authenticated subject remain mandatory even without optional JWTClaims.
type strictClaimsVerifier struct{ next tokenClaimsVerifier }

func (v strictClaimsVerifier) VerifyClaims(ctx context.Context, token string) (*jwt.Claims, error) {
	claims, err := v.next.VerifyClaims(ctx, token)
	if err != nil {
		return nil, err
	}
	if claims == nil || strings.TrimSpace(claims.Subject) == "" || claims.Issuer == "" || claims.ExpiresAt == nil || !claims.ExpiresAt.After(time.Now()) {
		return nil, errors.New("verified JWT requires subject, issuer and unexpired expiry")
	}
	return claims, nil
}
