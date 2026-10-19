package accessconfig

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"sync/atomic"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/viant/authz/oauth"
)

func scopedIdentityFixture(t *testing.T) (*verifiedIdentity, context.Context, *atomic.Int32) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	calls := &atomic.Int32{}
	p := &verifiedIdentity{tenant: "tenant", verifier: &ClaimsVerifier{key: func(*jwtlib.Token) (any, error) { calls.Add(1); return &key.PublicKey, nil }}}
	token, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{"iss": "https://issuer.example", "sub": "subject", "account_id": "account", "roles": []string{"editor"}, "exposures": []string{"studio"}, "membershipGroups": []string{"owners"}, "exp": time.Now().Add(time.Minute).Unix()}).SignedString(key)
	require.NoError(t, err)
	return p, oauth.WithBearer(context.Background(), token), calls
}
func TestVerifiedIdentityRequestScopesReuseOnlyVerifiedPrincipal(t *testing.T) {
	for _, metadata := range []bool{false, true} {
		t.Run(map[bool]string{false: "operation", true: "metadata"}[metadata], func(t *testing.T) {
			p, base, calls := scopedIdentityFixture(t)
			ctx, finish, err := p.beginIdentityScope(base, metadata)
			require.NoError(t, err)
			require.EqualValues(t, 1, calls.Load())
			require.Equal(t, metadata, p.MetadataReadActive(ctx))
			for i := 0; i < 10; i++ {
				facts, err := p.Resolve(ctx)
				require.NoError(t, err)
				account, err := p.Account(ctx, facts)
				require.NoError(t, err)
				_, _, err = p.AuthorityRevision(ctx, facts, account)
				require.NoError(t, err)
				principal, err := p.ResolvePrincipal(ctx)
				require.NoError(t, err)
				require.Equal(t, []string{"editor"}, principal.Facts.Roles)
				require.Equal(t, []string{"studio"}, principal.Facts.Exposures)
				require.Equal(t, []string{"owners"}, principal.MembershipGroups)
				principal.Facts.Roles[0] = "mutated"
				principal.Facts.Exposures[0] = "mutated"
				principal.MembershipGroups[0] = "mutated"
			}
			nested, end, err := p.BeginOperation(ctx)
			require.NoError(t, err)
			require.Equal(t, ctx, nested)
			require.NoError(t, end())
			require.NoError(t, finish())
			_, err = p.Resolve(ctx)
			require.Error(t, err)
			require.Error(t, finish())
			require.EqualValues(t, 1, calls.Load(), "scope close must not redo signature verification")
			_, err = p.Resolve(base)
			require.NoError(t, err)
			require.EqualValues(t, 2, calls.Load(), "a different request must verify independently")
		})
	}
}
func TestVerifiedIdentityScopeRejectsChangedBindingAndExpiredLifetime(t *testing.T) {
	for _, mode := range []string{"bearer", "provider", "expired", "cancelled", "closed", "invalid-entry"} {
		t.Run(mode, func(t *testing.T) {
			p, base, calls := scopedIdentityFixture(t)
			base, cancel := context.WithCancel(base)
			defer cancel()
			if mode == "invalid-entry" {
				_, _, err := p.BeginMetadataRead(oauth.WithBearer(base, "invalid"))
				require.Error(t, err)
				return
			}
			ctx, finish, err := p.BeginMetadataRead(base)
			require.NoError(t, err)
			selected := p
			switch mode {
			case "bearer":
				ctx = oauth.WithBearer(ctx, "different-token")
			case "provider":
				copy := *p
				selected = &copy
			case "expired":
				state := ctx.Value(identityScopeKey{}).(*identityScope)
				state.principal.Facts.ValidUntil = time.Now().Add(-time.Second)
			case "cancelled":
				cancel()
			case "closed":
				require.NoError(t, finish())
			}
			_, err = selected.ResolvePrincipal(ctx)
			require.Error(t, err)
			_, _, err = selected.BeginOperation(ctx)
			require.Error(t, err)
			require.False(t, selected.MetadataReadActive(ctx))
			require.EqualValues(t, 1, calls.Load(), "invalid scopes must deny without another verification")
			if mode == "expired" || mode == "cancelled" || mode == "closed" {
				require.Error(t, finish())
			} else {
				require.NoError(t, finish())
			}
		})
	}
}
