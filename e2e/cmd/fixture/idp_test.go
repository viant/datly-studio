package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/scy"
	"github.com/viant/scy/auth/jwt/signer"
)

type userInfoTransport func(*http.Request) (*http.Response, error)

func (f userInfoTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUserInfoOwnsPermissionsForSCYIdentity(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	public, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	for name, block := range map[string]*pem.Block{
		"private.pem": {Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)},
		"public.pem":  {Type: "PUBLIC KEY", Bytes: public},
	} {
		if err := os.WriteFile(filepath.Join(dir, name), pem.EncodeToMemory(block), 0600); err != nil {
			t.Fatal(err)
		}
	}
	signing := signer.New(&signer.Config{RSA: scy.NewResource(nil, filepath.Join(dir, "private.pem"), "")})
	if err := signing.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	token, err := signing.Create(time.Hour, map[string]any{
		"sub": "viewer", "iss": "studio-e2e", "aud": "studio-e2e", "user_id": 3, "account_id": 42,
		"roles": []string{"admin"}, "features": []string{"EXPOSE_ADMIN"},
		"entityPermissions": []map[string]any{{"type": "customer", "id": "10", "permissions": []string{"*"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := newIDP(dir)
	if err != nil {
		t.Fatal(err)
	}
	info := map[string]any{"subject": "viewer", "userId": 3, "accountId": 42, "roles": []string{"viewer"}, "features": []string{"EXPOSE_READ"}, "entityPermissions": []any{}, "evaluatedAt": time.Now().UTC(), "validUntil": time.Now().Add(time.Minute).UTC()}
	provider.client.Transport = userInfoTransport(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			t.Fatal("user-info credential missing")
		}
		body, err := json.Marshal(map[string]any{"status": "ok", "info": info})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body))), Header: make(http.Header)}, err
	})
	ctx := oauth.WithBearer(context.Background(), token)
	facts, err := provider.Resolve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(facts.Roles) != 1 || facts.Roles[0] != "viewer" || len(facts.Exposures) != 1 || facts.Exposures[0] != "EXPOSE_READ" || len(facts.EntityPermissions) != 0 {
		t.Fatalf("permissions must come only from user-info: %+v", facts)
	}
	for field, value := range map[string]any{"subject": "other", "userId": 99, "accountId": 99} {
		original := info[field]
		info[field] = value
		if _, err := provider.Resolve(ctx); !errors.Is(err, authz.ErrDenied) {
			t.Fatalf("mismatched %s accepted: %v", field, err)
		}
		info[field] = original
	}
}
