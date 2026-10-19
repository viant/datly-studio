package codecs

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	jwtlib "github.com/golang-jwt/jwt/v5"
	nativeauth "github.com/viant/datly/runtime/auth"
	"github.com/viant/scy"
	"github.com/viant/scy/auth/jwt"
	"github.com/viant/scy/auth/jwt/verifier"
	xcodec "github.com/viant/xdatly/codec"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestConfiguredJWTCodecVerifiesAndFailsClosed(t *testing.T) {
	ctx := context.Background()
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
	auth, err := nativeauth.New(ctx, &nativeauth.Config{JWTValidator: &verifier.Config{RSA: []*scy.Resource{{URL: path}}}, ClaimPolicy: &nativeauth.ClaimPolicy{Issuer: "test", Audience: "studio", RequireSubject: true}})
	if err != nil {
		t.Fatal(err)
	}
	cfg := &xcodec.Config{Body: "JwtClaim", SourceType: reflect.TypeFor[string](), DestinationType: reflect.TypeFor[*jwt.Claims]()}
	if _, err := FromContext(ctx).New(cfg); err == nil {
		t.Fatal("JWT accepted without verifier")
	}
	codec, err := FromContext(WithFactory(ctx, New(auth))).New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		issuer string
		valid  bool
	}{{"test", true}, {"wrong", false}} {
		token, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{"iss": test.issuer, "aud": "studio", "sub": "reader", "exp": time.Now().Add(time.Minute).Unix()}).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		value, err := codec.Value(ctx, "Bearer "+token)
		if test.valid {
			if err != nil || value.(*jwt.Claims).Subject != "reader" {
				t.Fatalf("verified claims: %v %v", value, err)
			}
		} else if err == nil {
			t.Fatal("wrong issuer accepted")
		}
	}
	for _, raw := range []string{"", "Bearer invalid"} {
		if _, err := codec.Value(ctx, raw); err == nil {
			t.Fatal("invalid credential accepted")
		}
	}
	cfg.Args = []string{"unsafe"}
	if _, err := New(auth).New(cfg); err == nil {
		t.Fatal("JWT transformation arguments accepted")
	}
}
