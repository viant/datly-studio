package sdk

import "context"

// Principal is the authenticated Studio identity made available to an SDK
// authorizer. HTTP adapters set it only after their mode-specific trust check.
type Principal struct {
	Subject     string
	Development bool
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
