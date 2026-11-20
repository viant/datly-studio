package accessprovider

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/viant/authz/oauth"
	"github.com/viant/scy"
	"github.com/viant/scy/auth/jwt/verifier"
)

func TestHTTPProvidersReuseSharedVerifierAndReadFreshFacts(t *testing.T) {
	for _, skip := range []bool{false, true} {
		t.Run(fmt.Sprint(skip), func(t *testing.T) { testHTTPProvidersAudiencePolicy(t, skip) })
	}
}
func testHTTPProvidersAudiencePolicy(t *testing.T, skip bool) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	public, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "public.pem")
	if err = os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: public}), 0600); err != nil {
		t.Fatal(err)
	}
	var reads atomic.Int32
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		if r.Header.Get("Authorization") == "" || r.Header.Get("Cache-Control") != "no-cache" {
			http.Error(w, "credential and fresh facts required", 401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"subject": "synthetic", "accountId": "42"})
	}))
	defer endpoint.Close()
	settings := Config{Issuer: endpoint.URL, Audience: "studio", SkipAudienceCheck: skip, JWTValidator: &verifier.Config{RSA: []*scy.Resource{{URL: path}}}}
	identity := oauth.HTTPIdentityConfig{UserInfoURL: endpoint.URL, TenantForAllAccounts: "studio", Mapping: oauth.HTTPIdentityMapping{Subject: "subject", AccountID: "accountId", JWTAccountID: "accountId"}}
	providers, err := HTTPProviders(context.Background(), settings, identity, nil)
	if err != nil {
		t.Fatal(err)
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"iss": endpoint.URL, "aud": "studio", "sub": "synthetic", "accountId": "42", "exp": time.Now().Add(time.Minute).Unix()}).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		facts, err := providers.Identity.Resolve(oauth.WithBearer(context.Background(), token))
		if err != nil || facts.Subject != "synthetic" || facts.Tenant != "studio" {
			t.Fatalf("verified user-info facts: %v", err)
		}
	}
	if reads.Load() != 2 {
		t.Fatal("mutable facts were cached by default")
	}
	resourceToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"iss": endpoint.URL, "aud": "https://resource.example/mcp", "sub": "synthetic", "accountId": "42", "exp": time.Now().Add(time.Minute).Unix()}).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	_, err = providers.Identity.Resolve(oauth.WithBearer(context.Background(), resourceToken))
	if (err == nil) != skip {
		t.Fatalf("HTTP audience policy skip=%v: %v", skip, err)
	}
	claimsVerifier, err := NewClaimsVerifier(settings)
	if err != nil {
		t.Fatal(err)
	}
	_, err = claimsVerifier.Verify(oauth.WithBearer(context.Background(), resourceToken))
	if (err == nil) != skip {
		t.Fatalf("claims audience policy skip=%v: %v", skip, err)
	}
	standalone, err := New(context.Background(), settings, nil)
	if err != nil {
		t.Fatal(err)
	}
	standaloneToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"iss": endpoint.URL, "aud": "https://resource.example/mcp", "sub": "synthetic", "accountId": "42", "tenant": "studio", "exp": time.Now().Add(time.Minute).Unix()}).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	_, err = standalone.Resolve(oauth.WithBearer(context.Background(), standaloneToken))
	if (err == nil) != skip {
		t.Fatalf("provider audience policy skip=%v: %v", skip, err)
	}
	identity.SkipAudienceCheck = true
	if _, err = HTTPProviders(context.Background(), settings, identity, nil); err == nil {
		t.Fatal("separate identity audience policy accepted")
	}
	identity.SkipAudienceCheck = false
	identity.Issuer = "repeated-source"
	if _, err = HTTPProviders(context.Background(), settings, identity, nil); err == nil {
		t.Fatal("separate identity verifier configuration accepted")
	}
}
