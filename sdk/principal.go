package sdk

import (
	"context"
	"errors"
	"strings"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// Principal is the authenticated Studio identity made available to an SDK
// authorizer. HTTP adapters set it only after their mode-specific trust check.
type Principal struct {
	Subject     string
	Development bool
}

// SystemPrincipal identifies an explicit internal Studio maintenance action.
// It is never substituted by PrincipalFromContext for an unauthenticated call.
func SystemPrincipal() Principal { return Principal{Subject: "system:datly-studio"} }

// SystemCredentialProvider supplies a provider-issued, already-verified
// service JWT. It is optional for in-process maintenance; callers that send
// an outbound bearer must configure a provider.
type SystemCredentialProvider func(context.Context) (VerifiedCredential, error)

// WithSystemIdentity binds an explicit system principal. With a provider it
// also binds a verified service JWT; without one it binds only trusted
// in-process principal claims, never an outbound bearer.
func WithSystemIdentity(ctx context.Context, provider SystemCredentialProvider) (context.Context, error) {
	principal := SystemPrincipal()
	// A maintenance transition must never inherit the triggering user's bearer.
	ctx = context.WithValue(ctx, credentialKey{}, VerifiedCredential{})
	if provider == nil {
		return WithPrincipal(ctx, principal), nil
	}
	credential, err := provider(ctx)
	if err != nil {
		return nil, err
	}
	claims, ok := credential.Claims.(jwtv5.Claims)
	if !ok || !strings.HasPrefix(credential.Bearer, "Bearer ") {
		return nil, errors.New("system credential must contain a verified bearer and JWT claims")
	}
	subject, err := claims.GetSubject()
	if err != nil || subject != principal.Subject {
		return nil, errors.New("system JWT subject does not match system principal")
	}
	expiry, err := claims.GetExpirationTime()
	if err != nil || expiry == nil || !expiry.After(time.Now()) {
		return nil, errors.New("system JWT is expired or missing expiry")
	}
	return WithVerifiedCredential(WithPrincipal(ctx, principal), credential), nil
}

type principalKey struct{}
type credentialKey struct{}

// VerifiedCredential is server-only authentication evidence. It must never be
// serialized into SDK responses or exposed to Forge.
type VerifiedCredential struct {
	Bearer string
	Claims any
}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(Principal)
	return principal, ok && principal.Subject != ""
}

func WithVerifiedCredential(ctx context.Context, credential VerifiedCredential) context.Context {
	return context.WithValue(ctx, credentialKey{}, credential)
}

func VerifiedCredentialFromContext(ctx context.Context) (VerifiedCredential, bool) {
	credential, ok := ctx.Value(credentialKey{}).(VerifiedCredential)
	return credential, ok && credential.Claims != nil
}
