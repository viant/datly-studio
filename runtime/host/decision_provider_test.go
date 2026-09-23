package host

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

	"github.com/viant/datly-studio/sdk/access"
	"go.yaml.in/yaml/v3"
	_ "modernc.org/sqlite"
)

type injectedDecisions struct{ calls int }

func (d *injectedDecisions) Evaluate(context.Context, access.Request, access.Document, access.Facts) (access.Decision, error) {
	d.calls++
	return access.Decision{}, nil
}

type injectedFacts struct{}

func (injectedFacts) Resolve(context.Context) (access.Facts, error) {
	return access.Facts{Subject: "alice", Tenant: "one", Issuer: "issuer", Roles: []string{"reader"}, ValidUntil: time.Now().Add(time.Minute)}, nil
}

type injectedStore struct{ doc access.Document }

func (s injectedStore) Get(context.Context, access.Resource) (access.Document, error) {
	return s.doc, nil
}
func (injectedStore) Replace(context.Context, access.Document, int64, string) (access.Document, error) {
	return access.Document{}, access.ErrDenied
}

func TestNativeResourceAccessUsesInjectedProvider(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(t.TempDir(), "access.pem")
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: encoded}), 0600); err != nil {
		t.Fatal(err)
	}
	decisions := &injectedDecisions{}
	config := Config{
		HTTP: Listener{Address: "127.0.0.1:0"}, MCP: Listener{Address: "127.0.0.1:0"},
		Authentication: Authentication{DefaultMode: "public"},
		Studio:         Studio{Driver: "sqlite", DSN: ":memory:"}, Admin: Admin{Token: "test-token"},
		Access:           &ResourceAccessConfig{Tenant: "one", Issuer: "issuer", Audience: "runtime", PublicKeyFile: keyFile},
		DecisionProvider: decisions,
	}
	encodedConfig, err := yaml.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if string(encodedConfig) == "" || containsProviderField(encodedConfig) {
		t.Fatal("injected provider appeared in YAML")
	}
	service, err := New(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.Close(context.Background()) })
	if service.resourceAccess == nil || service.resourceAccess.Decisions != decisions {
		t.Fatal("native access service did not retain the injected decision provider")
	}
	r := access.Resource{Kind: "component", ID: "report", Version: "1", Tenant: "one"}
	service.resourceAccess.Store = injectedStore{doc: access.Document{Resource: r, Revision: 1, Policies: map[string]access.Policy{"execute": {Mode: "protected", Rule: &access.Rule{Kind: "role", Value: "reader"}}}}}
	service.resourceAccess.Provider = injectedFacts{}
	if _, err := service.resourceAccess.Authorize(context.Background(), access.Request{Resource: r, Action: "execute"}); err != nil {
		t.Fatal(err)
	}
	if decisions.calls != 1 {
		t.Fatalf("remote provider called %d times", decisions.calls)
	}
}

func containsProviderField(encoded []byte) bool {
	var fields map[string]any
	if err := yaml.Unmarshal(encoded, &fields); err != nil {
		return true
	}
	_, present := fields["DecisionProvider"]
	return present
}
