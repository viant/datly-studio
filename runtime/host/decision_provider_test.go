package host

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/viant/authz"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/runtime/accesscontext"
	rhandler "github.com/viant/datly/runtime/handler"
	"github.com/viant/datly/runtime/registry"
	"github.com/viant/datly/spec"
	"go.yaml.in/yaml/v3"
	_ "modernc.org/sqlite"
)

type injectedDecisions struct{ calls int }

func (d *injectedDecisions) Evaluate(context.Context, authz.Request, authz.Document, authz.Facts) (authz.Decision, error) {
	d.calls++
	return authz.Decision{}, nil
}

type injectedFacts struct{}

func (injectedFacts) Resolve(context.Context) (authz.Facts, error) {
	return authz.Facts{Subject: "alice", Tenant: "one", Issuer: "issuer", Roles: []string{"reader"}, ValidUntil: time.Now().Add(time.Minute)}, nil
}

type countingFacts struct{ calls int }

func (p *countingFacts) Resolve(context.Context) (authz.Facts, error) {
	p.calls++
	return authz.Facts{Subject: "alice", Tenant: "one", Issuer: "issuer", Roles: []string{"reader"},
		Entities: []authz.Entity{{Type: "project", ID: "101"}}, ValidUntil: time.Now().Add(time.Minute)}, nil
}

type injectedStore struct{ doc authz.Document }

func (s injectedStore) Get(context.Context, authz.Resource) (authz.Document, error) {
	return s.doc, nil
}
func (injectedStore) Replace(context.Context, authz.Document, int64, string) (authz.Document, error) {
	return authz.Document{}, authz.ErrDenied
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
	userInfo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"status":"ok","info":{"uid":"alice","subject":"alice","userId":7,"accountId":21,"roles":["reader"],"features":["export"],"entityPermissions":[]}}`))
	}))
	defer userInfo.Close()
	config := Config{
		HTTP: Listener{Address: "127.0.0.1:0"}, MCP: Listener{Address: "127.0.0.1:0"},
		Authentication: Authentication{DefaultMode: "public"},
		Studio:         Studio{Driver: "sqlite", DSN: ":memory:"}, Admin: Admin{Token: "test-token"},
		Access:           &ResourceAccessConfig{Tenant: "one", Issuer: "issuer", Audience: "runtime", PublicKeyFile: keyFile, UserInfoURL: userInfo.URL},
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
	identityToken, err := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{
		"iss": "issuer", "aud": "runtime", "sub": "alice", "user_id": 7, "account_id": 21,
		"exp": time.Now().Add(59 * time.Minute).Unix(),
	}).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	facts, err := service.resourceAccess.Provider.Resolve(oauth.WithBearer(context.Background(), identityToken))
	if err != nil || facts.Tenant != "21" || len(facts.Roles) != 1 || facts.Roles[0] != "reader" || len(facts.Exposures) != 1 || facts.Exposures[0] != "export" {
		t.Fatalf("runtime identity facts=%+v error=%v", facts, err)
	}
	r := authz.Resource{Kind: "component", ID: "report", Version: "1", Tenant: "one"}
	service.resourceAccess.Store = injectedStore{doc: authz.Document{Resource: r, Revision: 1, Policies: map[string]authz.Policy{"execute": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}}}}}
	service.resourceAccess.Provider = injectedFacts{}
	if _, err := service.resourceAccess.Authorize(context.Background(), authz.Request{Resource: r, Action: "execute"}); err != nil {
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

func TestBoundAccessContextUsesOneVerifiedFactsSnapshot(t *testing.T) {
	resource := authz.Resource{Kind: "component", ID: "tasks", Version: "1", Tenant: "one"}
	provider := &countingFacts{}
	service := &Service{config: Config{Access: &ResourceAccessConfig{Tenant: "one"}}, resourceAccess: &authz.Service{
		Store: injectedStore{doc: authz.Document{Resource: resource, Revision: 1, Policies: map[string]authz.Policy{
			"execute": {Mode: "protected", Rule: &authz.Rule{Kind: "role", Value: "reader"}, EntityType: "project"},
		}}}, Provider: provider,
	}}
	component := &spec.Component{Key: spec.Key{Kind: spec.KindComponent, Scope: "example.com/tasks", Name: "Tasks"}, Parameters: []*spec.Parameter{{
		Name: "Auth", Source: spec.BindSource{Kind: string(spec.KindComponent), Name: "GET:/_studio/access/context/tasks/project"},
	}}}
	contexts, binds, err := service.accessContexts([]*registry.RegisteredComponent{{Component: component}}, map[spec.Key]string{component.Key: "tasks"}, map[string]int{"tasks": 1})
	if err != nil || !binds["tasks"] || len(contexts) != 1 {
		t.Fatalf("contexts=%d binds=%v error=%v", len(contexts), binds, err)
	}
	actual, err := contexts[0].Handler.Execute(context.Background(), rhandler.Invocation{Input: &accesscontext.Input{}})
	output, ok := actual.(*accesscontext.Output)
	if err != nil || !ok || output.Scope == nil || len(output.Scope.IDs) != 1 || output.Scope.IDs[0] != "101" || provider.calls != 1 {
		t.Fatalf("output=%+v provider calls=%d error=%v", actual, provider.calls, err)
	}
}
