package datatest

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	runtimeauth "github.com/viant/datly/runtime/auth"
	"github.com/viant/scy"
	"github.com/viant/scy/auth/jwt/verifier"
)

// JWTFixture supplies the same RSA verifier/signing arrangement used by the
// Platform E2E suite. Its tokens exercise Datly's JwtClaim codec end to end.
type JWTFixture struct {
	private *rsa.PrivateKey
	Factory *runtimeauth.Service
}

func NewJWTFixture(t testing.TB) *JWTFixture {
	t.Helper()
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	public, err := x509.MarshalPKIXPublicKey(&private.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	factory, err := runtimeauth.New(context.Background(), &runtimeauth.Config{JWTValidator: &verifier.Config{RSA: []*scy.Resource{{URL: "datly-studio-test-key", Data: pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: public})}}}})
	if err != nil {
		t.Fatal(err)
	}
	return &JWTFixture{private: private, Factory: factory}
}

func (f *JWTFixture) Bearer(t testing.TB, subject string) string {
	t.Helper()
	token := jwtv5.NewWithClaims(jwtv5.SigningMethodRS256, jwtv5.MapClaims{
		"sub": subject, "user_id": 1, "exp": time.Now().Add(time.Hour).Unix(),
	})
	encoded, err := token.SignedString(f.private)
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + encoded
}
