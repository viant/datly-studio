package studioapi

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/viant/scy"
	"github.com/viant/scy/auth/jwt/signer"
	"github.com/viant/scy/auth/jwt/verifier"
)

func TestAuthenticatedHostUsesSCYRSAOrCertificateVerification(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	public, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: public})
	dir := t.TempDir()
	privatePath, publicPath := filepath.Join(dir, "private.pem"), filepath.Join(dir, "public.pem")
	if err = os.WriteFile(privatePath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(publicPath, publicPEM, 0600); err != nil {
		t.Fatal(err)
	}
	signing := signer.New(&signer.Config{RSA: scy.NewResource(nil, privatePath, "")})
	if err = signing.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	token, err := signing.Create(time.Hour, map[string]any{"sub": "owner", "iss": "studio-e2e", "aud": "studio-e2e", "user_id": 7, "account_id": 21})
	if err != nil {
		t.Fatal(err)
	}
	config, err := jwtVerificationConfig("", publicPath)
	if err != nil {
		t.Fatal(err)
	}
	rsaVerifier := verifier.New(config)
	if err = rsaVerifier.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	keys, err := rsaVerifier.PublicKeys()
	if err != nil {
		t.Fatal(err)
	}
	certificates := []map[string]string{}
	for kid, key := range keys {
		certificates = append(certificates, map[string]string{"kty": "RSA", "use": "sig", "alg": "RS256", "kid": kid, "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": "AQAB"})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("credential forwarded to certificates")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": certificates})
	}))
	defer server.Close()
	for _, mode := range []struct{ name, cert, key string }{{"rsa", "", publicPath}, {"certificate", server.URL, ""}} {
		t.Run(mode.name, func(t *testing.T) {
			cfg, err := jwtVerificationConfig(mode.cert, mode.key)
			if err != nil {
				t.Fatal(err)
			}
			service := verifier.New(cfg)
			if err = service.Init(context.Background()); err != nil {
				t.Fatal(err)
			}
			claims, err := service.VerifyClaims(context.Background(), token)
			if err != nil {
				t.Fatal(err)
			}
			if claims.Subject != "owner" || claims.UserID != 7 || claims.AccountId != 21 || !claims.VerifyAudience("studio-e2e", true) {
				t.Fatal("verified identity changed")
			}
			sessionKey := base64.StdEncoding.EncodeToString(make([]byte, 32))
			if _, _, _, err = resolveAuthenticatedConfig(mode.cert, "studio-e2e", "studio-e2e", sessionKey, mode.key); err != nil {
				t.Fatal(err)
			}
		})
	}
	if _, err = jwtVerificationConfig(server.URL, publicPath); err == nil {
		t.Fatal("ambiguous verification sources accepted")
	}
}

func TestSharedJWTValidatorPreservesSCYResources(t *testing.T) {
	for _, config := range []*verifier.Config{
		{CertURL: "https://auth.example.com/certs"},
		{RSA: []*scy.Resource{{URL: "file:///one.enc", Key: "blowfish://default"}, {URL: "file:///two.pem"}}},
	} {
		got, err := configuredJWTVerification(config, "", "")
		if err != nil || got != config {
			t.Fatalf("shared verifier was not preserved: %v", err)
		}
		if _, err := configuredJWTVerification(config, "https://other.example/certs", ""); err == nil {
			t.Fatal("conflicting certificate override accepted")
		}
		if _, err := configuredJWTVerification(config, "", "file:///other.pem"); err == nil {
			t.Fatal("conflicting RSA override accepted")
		}
	}
	for _, config := range []*verifier.Config{
		{}, {RSA: []*scy.Resource{nil}}, {RSA: []*scy.Resource{{}}},
		{CertURL: "https://auth.example/certs", RSA: []*scy.Resource{{URL: "file:///public.pem"}}},
	} {
		if _, err := configuredJWTVerification(config, "", ""); err == nil {
			t.Fatal("invalid shared verifier accepted")
		}
	}
}
