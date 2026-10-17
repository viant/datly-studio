package accessconfig

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	accessoauth "github.com/viant/authz/oauth"
	"github.com/viant/scy"
	"github.com/viant/scy/auth/jwt/verifier"
)

func TestSharedRSAResourceVerifiesClaimsAndDeniesWrongAuthority(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	public, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "public.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: public}), 0600); err != nil {
		t.Fatal(err)
	}
	config := Config{Issuer: "https://synthetic.example", Audience: "studio", JWTValidator: &verifier.Config{RSA: []*scy.Resource{{URL: path}}}}
	provider, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, issuer, audience string
		valid                  bool
	}{
		{"allowed", config.Issuer, config.Audience, true},
		{"wrong issuer", "https://other.example", config.Audience, false},
		{"wrong audience", config.Issuer, "other-app", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			claims := accessoauth.Claims{RegisteredClaims: jwtlib.RegisteredClaims{Issuer: test.issuer, Audience: []string{test.audience}, Subject: "synthetic-owner", ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(time.Minute))}, Tenant: "synthetic"}
			token, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims).SignedString(key)
			if err != nil {
				t.Fatal(err)
			}
			facts, err := provider.Resolve(accessoauth.WithBearer(context.Background(), token))
			if test.valid {
				if err != nil || facts.Subject != "synthetic-owner" {
					t.Fatalf("facts: %+v %v", facts, err)
				}
			} else if err == nil {
				t.Fatal("wrong authority accepted")
			}
		})
	}
	if _, err := provider.Resolve(accessoauth.WithBearer(context.Background(), "forged")); err == nil {
		t.Fatal("forged credential accepted")
	}
	config.PublicKeyFile = path
	if _, err := New(config); err == nil {
		t.Fatal("competing key configurations accepted")
	}
}

func TestSharedVerifierRejectsInvalidProfiles(t *testing.T) {
	for _, config := range []*verifier.Config{
		{}, {RSA: []*scy.Resource{nil}},
		{RSA: []*scy.Resource{{URL: "file:///unused"}}, CertURL: "https://synthetic.example/certs"},
		{HMAC: &scy.Resource{URL: "file:///unused"}},
	} {
		if _, err := configuredKeyfunc(config, "", ""); err == nil {
			t.Fatal("invalid profile accepted")
		}
	}
}
