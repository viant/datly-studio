package httptransport

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"strings"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/viant/datly-studio/sdk"
	scyjwt "github.com/viant/scy/auth/jwt"
)

// DevelopmentJWT owns one ephemeral signing key per local gateway process.
// It is not a production issuer and its tokens must not be used off loopback.
type DevelopmentJWT struct {
	key      *rsa.PrivateKey
	issuer   string
	audience string
}

func NewDevelopmentJWT(issuer, audience string) (*DevelopmentJWT, error) {
	if strings.TrimSpace(issuer) == "" || strings.TrimSpace(audience) == "" {
		return nil, errors.New("development JWT issuer and audience are required")
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return &DevelopmentJWT{key: key, issuer: issuer, audience: audience}, nil
}

// Credential returns a freshly signed and locally verified five-minute JWT.
func (p *DevelopmentJWT) Credential(_ context.Context, subject string) (sdk.VerifiedCredential, error) {
	if p == nil || p.key == nil || strings.TrimSpace(subject) == "" {
		return sdk.VerifiedCredential{}, errors.New("development JWT signer and subject are required")
	}
	now := time.Now().UTC()
	claims := &scyjwt.Claims{RegisteredClaims: jwtv5.RegisteredClaims{Issuer: p.issuer,
		Audience: jwtv5.ClaimStrings{p.audience}, Subject: subject,
		IssuedAt: jwtv5.NewNumericDate(now), ExpiresAt: jwtv5.NewNumericDate(now.Add(5 * time.Minute))}}
	signed, err := jwtv5.NewWithClaims(jwtv5.SigningMethodRS256, claims).SignedString(p.key)
	if err != nil {
		return sdk.VerifiedCredential{}, err
	}
	verified := &scyjwt.Claims{}
	parsed, err := jwtv5.NewParser(jwtv5.WithValidMethods([]string{"RS256"}), jwtv5.WithIssuer(p.issuer),
		jwtv5.WithAudience(p.audience)).ParseWithClaims(signed, verified, func(*jwtv5.Token) (any, error) {
		return &p.key.PublicKey, nil
	})
	if err != nil || parsed == nil || !parsed.Valid || verified.Subject != subject {
		return sdk.VerifiedCredential{}, errors.New("development JWT self-verification failed")
	}
	return sdk.VerifiedCredential{Bearer: "Bearer " + signed, Claims: verified}, nil
}
