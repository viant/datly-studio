package accessconfig

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	accessoauth "github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/sdk"
)

func TestJWKSAuthorityRefreshesRotatedKeysWithoutTrustingUnknownIDs(t *testing.T) {
	first, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	second, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var current atomic.Int32
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		key, kid := &first.PublicKey, "first"
		if current.Load() == 1 {
			key, kid = &second.PublicKey, "second"
		}
		payload := map[string]any{"keys": []map[string]string{{
			"kty": "RSA", "kid": kid, "alg": "RS256", "use": "sig",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}}
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer server.Close()
	provider, err := New(Config{Issuer: "issuer", Audience: "audience", CertURL: server.URL + "/jwks"})
	if err != nil {
		t.Fatal(err)
	}
	sign := func(key *rsa.PrivateKey, kid string) string {
		t.Helper()
		claims := accessoauth.Claims{RegisteredClaims: jwtlib.RegisteredClaims{
			Issuer: "issuer", Audience: []string{"audience"}, Subject: "alice",
			ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(time.Minute)),
		}, Tenant: "one", Roles: []string{"reader"}}
		token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims)
		token.Header["kid"] = kid
		value, err := token.SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	resolve := func(bearer string) error {
		ctx := sdk.WithVerifiedCredential(context.Background(), sdk.VerifiedCredential{Bearer: bearer, Claims: struct{}{}})
		_, err := provider.Resolve(ctx)
		return err
	}
	old := sign(first, "first")
	if err := resolve(old); err != nil {
		t.Fatal(err)
	}
	if err := resolve(old); err != nil || requests.Load() != 1 {
		t.Fatalf("known key was fetched repeatedly: calls=%d err=%v", requests.Load(), err)
	}
	current.Store(1)
	if err := resolve(sign(second, "second")); err != nil || requests.Load() != 2 {
		t.Fatalf("rotated signing key was not fetched: calls=%d err=%v", requests.Load(), err)
	}
	if err := resolve(old); err == nil {
		t.Fatal("removed signing key remained trusted")
	}
	if err := resolve(sign(second, "unknown")); err == nil {
		t.Fatal("unknown key ID was accepted")
	}
}

func TestJWKSAuthorityRequiresTrustedEndpointAndOneKeySource(t *testing.T) {
	for _, config := range []Config{
		{Issuer: "issuer", Audience: "audience", CertURL: "http://idp.example/jwks"},
		{Issuer: "issuer", Audience: "audience", CertURL: "https://idp.example/jwks?token=x"},
		{Issuer: "issuer", Audience: "audience", CertURL: "https://idp.example/jwks", PublicKeyFile: "/tmp/key.pem"},
	} {
		if _, err := New(config); err == nil {
			t.Fatalf("unsafe authority source accepted: %+v", config)
		}
	}
}
