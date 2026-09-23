package oauth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/viant/datly-studio/sdk/access"
)

func TestVerifyIdentityAndFacts(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	provider, err := New(Config{Issuer: "https://identity.example", Audience: "studio-access", Algorithms: []string{"EdDSA"}, Keyfunc: func(*jwt.Token) (any, error) { return pub, nil }})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*Claims)
		denied bool
	}{
		{"valid", func(*Claims) {}, false},
		{"wrong issuer", func(c *Claims) { c.Issuer = "https://attacker.example" }, true},
		{"wrong audience", func(c *Claims) { c.Audience = []string{"other"} }, true},
		{"expired", func(c *Claims) { c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute)) }, true},
		{"missing expiry", func(c *Claims) { c.ExpiresAt = nil }, true},
		{"missing tenant", func(c *Claims) { c.Tenant = "" }, true},
		{"wildcard tenant", func(c *Claims) { c.Tenant = "*" }, true},
		{"invalid entity", func(c *Claims) { c.AllowedEntities = []access.Entity{{ID: "42"}} }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claims := Claims{RegisteredClaims: jwt.RegisteredClaims{Issuer: "https://identity.example", Subject: "alice", Audience: []string{"studio-access"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute))}, Tenant: "one", Roles: []string{"reader"}, Exposures: []string{"analytics"}, AllowedEntities: []access.Entity{{Type: "project", ID: "42"}}}
			tc.change(&claims)
			token, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims).SignedString(priv)
			if err != nil {
				t.Fatal(err)
			}
			facts, err := provider.Resolve(WithBearer(context.Background(), token))
			if (err != nil) != tc.denied {
				t.Fatalf("facts=%+v error=%v", facts, err)
			}
			if !tc.denied && (facts.Subject != "alice" || len(facts.Entities) != 1 || facts.Exposures[0] != "analytics") {
				t.Fatalf("lost claims: %+v", facts)
			}
		})
	}
	_, wrongKey, _ := ed25519.GenerateKey(rand.Reader)
	token, _ := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{"iss": "https://identity.example", "aud": "studio-access", "sub": "alice", "tenant": "one", "exp": time.Now().Add(time.Minute).Unix()}).SignedString(wrongKey)
	if _, err := provider.Resolve(WithBearer(context.Background(), token)); err == nil {
		t.Fatal("forged signature accepted")
	}
}
