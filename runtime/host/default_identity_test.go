package host

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/viant/scy"
	"github.com/viant/scy/auth/jwt/verifier"
)

func TestDefaultRuntimeIdentityBindsIssuerAudienceOnHTTPAndComponentPaths(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	verified := verifier.New(&verifier.Config{RSA: []*scy.Resource{{URL: "runtime-identity-test",
		Data: pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: encoded})}}})
	if err := verified.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	identity := Authentication{DefaultMode: "required", Issuer: "https://identity.example", Audience: "studio-web"}
	service := &Service{verifier: verified, config: Config{Authentication: identity}}
	component := &Service{verifier: verified, config: Config{Authentication: Authentication{
		DefaultMode: identity.DefaultMode, Issuer: identity.Issuer, Audience: identity.Audience,
		Components: map[string]ComponentAuthentication{"private": {Provider: "named"}},
	}}}
	for _, candidate := range []struct {
		name, issuer, audience, subject string
		status                          int
	}{
		{"matching ID token", identity.Issuer, identity.Audience, "alice", http.StatusNoContent},
		{"other issuer", "https://other.example", identity.Audience, "alice", http.StatusUnauthorized},
		{"other audience", identity.Issuer, "other-client", "alice", http.StatusUnauthorized},
		{"missing subject", identity.Issuer, identity.Audience, "", http.StatusUnauthorized},
	} {
		t.Run(candidate.name, func(t *testing.T) {
			claims := jwtv5.MapClaims{"iss": candidate.issuer, "aud": candidate.audience,
				"sub": candidate.subject, "exp": time.Now().Add(time.Hour).Unix()}
			token, err := jwtv5.NewWithClaims(jwtv5.SigningMethodRS256, claims).SignedString(key)
			if err != nil {
				t.Fatal(err)
			}
			for name, middleware := range map[string]func(http.Handler) http.Handler{
				"default": service.authenticated, "component": component.componentAuthenticated,
			} {
				t.Run(name, func(t *testing.T) {
					next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
					request := httptest.NewRequest(http.MethodGet, "/records", nil)
					request.Header.Set("Authorization", "Bearer "+token)
					response := httptest.NewRecorder()
					middleware(next).ServeHTTP(response, request)
					if response.Code != candidate.status {
						t.Fatalf("status=%d want=%d", response.Code, candidate.status)
					}
				})
			}
		})
	}
}
