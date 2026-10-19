package accessconfig

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"github.com/viant/authz/gating"
	"github.com/viant/authz/oauth"
	"github.com/viant/scy"
	"github.com/viant/scy/auth/jwt/verifier"
)

func TestJWTValidatorOnlyExposesVerifiedClaimsAndServerTenant(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	public, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "public.pem")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: public}), 0600))
	config := Config{Tenant: "server-tenant", JWTValidator: &verifier.Config{RSA: []*scy.Resource{{URL: path}}}}
	claimsVerifier, err := NewClaimsVerifier(config)
	require.NoError(t, err)
	provider, err := New(config)
	require.NoError(t, err)
	for _, mode := range []string{"valid", "expired", "missing-expiry", "missing-subject", "tampered"} {
		t.Run(mode, func(t *testing.T) {
			claims := jwtlib.MapClaims{"iss": "https://signed.example/", "aud": "any-audience", "sub": "owner", "tenant": "untrusted-tenant-choice", "account_id": json.Number("9007199254740993"), "membershipGroups": []string{"owners"}, "exp": time.Now().Add(time.Minute).Unix()}
			switch mode {
			case "expired":
				claims["exp"] = time.Now().Add(-time.Minute).Unix()
			case "missing-expiry":
				delete(claims, "exp")
			case "missing-subject":
				delete(claims, "sub")
			}
			token, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims).SignedString(key)
			require.NoError(t, err)
			if mode == "tampered" {
				token = token[:len(token)-10] + "tamperedxx"
			}
			ctx := oauth.WithBearer(context.Background(), token)
			verified, err := claimsVerifier.Verify(ctx)
			if mode != "valid" {
				require.Error(t, err)
				require.Nil(t, verified)
				_, err = provider.Resolve(ctx)
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, json.Number("9007199254740993"), verified["account_id"])
			principal, err := provider.(gating.PrincipalResolver).ResolvePrincipal(ctx)
			require.NoError(t, err)
			require.Equal(t, "server-tenant", principal.Facts.Tenant)
			require.Equal(t, "https://signed.example/", principal.Facts.Issuer)
			require.Equal(t, "9007199254740993", principal.AccountID)
			require.Equal(t, []string{"owners"}, principal.MembershipGroups)
		})
	}
	// Optional legacy restrictions still reject a signed token for another issuer.
	restricted := config
	restricted.Issuer = "https://expected.example/"
	v, err := NewClaimsVerifier(restricted)
	require.NoError(t, err)
	token, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{"iss": "https://other.example/", "sub": "owner", "exp": time.Now().Add(time.Minute).Unix()}).SignedString(key)
	require.NoError(t, err)
	_, err = v.Verify(oauth.WithBearer(context.Background(), token))
	require.Error(t, err)
}
