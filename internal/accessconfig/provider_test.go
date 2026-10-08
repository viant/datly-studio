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
	access "github.com/viant/authz"
	accessoauth "github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/sdk"
)

func TestEnvironmentProviderRequiresCompleteAuthority(t *testing.T) {
	for _, name := range []string{"STUDIO_ACCESS_ISSUER", "STUDIO_ACCESS_AUDIENCE", "STUDIO_ACCESS_PUBLIC_KEY_FILE", "STUDIO_ACCESS_USER_INFO_URL"} {
		t.Setenv(name, "")
	}
	provider, err := FromEnvironment()
	if err != nil || provider != nil {
		t.Fatalf("absent config: provider=%T err=%v", provider, err)
	}
	t.Setenv("STUDIO_ACCESS_ISSUER", "https://trusted.example")
	if _, err := FromEnvironment(); err == nil {
		t.Fatal("partial configuration accepted")
	}
	t.Setenv("STUDIO_ACCESS_AUDIENCE", "studio")
	path := filepath.Join(t.TempDir(), "public.pem")
	if err := os.WriteFile(path, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STUDIO_ACCESS_PUBLIC_KEY_FILE", path)
	if _, err := FromEnvironment(); err == nil {
		t.Fatal("invalid public key accepted")
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: raw}), 0600); err != nil {
		t.Fatal(err)
	}
	provider, err = FromEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	claims := accessoauth.Claims{RegisteredClaims: jwtlib.RegisteredClaims{Issuer: "https://trusted.example", Audience: []string{"studio"}, Subject: "alice", ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(time.Minute))}, Tenant: "tenant", Roles: []string{"reader"}}
	bearer, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sdk.WithVerifiedCredential(context.Background(), sdk.VerifiedCredential{Bearer: bearer, Claims: struct{}{}})
	facts, err := provider.Resolve(ctx)
	if err != nil || facts.Subject != "alice" || facts.Tenant != "tenant" {
		t.Fatalf("verified facts: %+v err=%v", facts, err)
	}
	claims.Issuer = "https://untrusted.example"
	bearer, err = jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	ctx = sdk.WithVerifiedCredential(context.Background(), sdk.VerifiedCredential{Bearer: bearer, Claims: struct{}{}})
	if _, err := provider.Resolve(ctx); err == nil {
		t.Fatal("wrong issuer accepted")
	}
	t.Setenv("STUDIO_ACCESS_USER_INFO_URL", "http://untrusted.example")
	if _, err := FromEnvironment(); err == nil {
		t.Fatal("insecure remote user-info endpoint accepted")
	}
}

func TestUserInfoProviderRequiresExplicitHostBinding(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "public.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: raw}), 0600); err != nil {
		t.Fatal(err)
	}
	config := Config{Issuer: "issuer", Audience: "studio", PublicKeyFile: path, UserInfoURL: "https://identity.example/userinfo"}
	if provider, err := New(config); err == nil || provider != nil {
		t.Fatalf("user-info config without host binding accepted: provider=%T err=%v", provider, err)
	}
	called := false
	provider, err := NewWithUserInfo(config, func(issuer, audience, endpoint string, keyfunc jwtlib.Keyfunc) (access.Provider, error) {
		called = true
		if issuer != config.Issuer || audience != config.Audience || endpoint != config.UserInfoURL || keyfunc == nil {
			t.Fatalf("factory received unexpected binding: issuer=%q audience=%q endpoint=%q keyfunc=%v", issuer, audience, endpoint, keyfunc != nil)
		}
		return fixedProvider{}, nil
	})
	if err != nil || !called {
		t.Fatalf("injected provider was not used: called=%v provider=%T err=%v", called, provider, err)
	}
	if provider, err := NewWithUserInfo(config, func(string, string, string, jwtlib.Keyfunc) (access.Provider, error) {
		return nil, nil
	}); err == nil || provider != nil {
		t.Fatalf("nil host provider accepted: provider=%T err=%v", provider, err)
	}
}

type fixedProvider struct{}

func (fixedProvider) Resolve(context.Context) (access.Facts, error) { return access.Facts{}, nil }
