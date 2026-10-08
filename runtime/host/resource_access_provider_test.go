package host

import (
	"context"
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	access "github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"net/http"
	"os"
	"testing"
	"time"
)

type injectedResourceIdentity struct{ calls int }

func (p *injectedResourceIdentity) Resolve(context.Context) (access.Facts, error) {
	p.calls++
	return access.Facts{}, access.ErrDenied
}
func TestRuntimeResourceAccessUsesInjectedPrivateBindingWithoutInterpretingDTO(t *testing.T) {
	provider := &injectedResourceIdentity{}
	config := Config{HTTP: Listener{Address: "127.0.0.1:0"}, MCP: Listener{Address: "127.0.0.1:0"}, Authentication: Authentication{DefaultMode: "public"}, Studio: Studio{Driver: "sqlite", DSN: "file:test.db"}, Admin: Admin{Token: "owned-token"}, Access: &ResourceAccessConfig{Tenant: "team", Provider: provider, UserInfoURL: "https://host-owned.example/profile"}}
	if err := config.Validate(); err != nil {
		t.Fatal(err)
	}
	service := &Service{config: config}
	if err := service.initResourceAccess(); err != nil {
		t.Fatal(err)
	}
	if service.resourceAccess.Provider != provider {
		t.Fatal("injected authority replaced by an implicit wire adapter")
	}
	if _, err := service.resourceAccess.Provider.Resolve(context.Background()); err != access.ErrDenied || provider.calls != 1 {
		t.Fatalf("injected provider not used: calls=%d err=%v", provider.calls, err)
	}
}
func TestRuntimeResourceAccessRejectsUnboundPrivateUserInfo(t *testing.T) {
	service := &Service{config: Config{Access: &ResourceAccessConfig{Tenant: "team", Issuer: "issuer", Audience: "audience", UserInfoURL: "https://host-owned.example/profile"}}}
	if err := service.initResourceAccess(); err == nil {
		t.Fatal("unbound private identity DTO activated")
	}
}

// Fixture-owned remote facts demonstrate injection without encoding an IdP DTO.
type fixtureRemoteFacts struct {
	verifier *oauth.Provider
	endpoint string
}

func (p fixtureRemoteFacts) Resolve(ctx context.Context) (access.Facts, error) {
	identity, err := p.verifier.Resolve(ctx)
	if err != nil {
		return access.Facts{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint, nil)
	if err != nil {
		return access.Facts{}, err
	}
	request.Header.Set("Authorization", "Bearer "+oauth.Bearer(ctx))
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return access.Facts{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return access.Facts{}, access.ErrDenied
	}
	var facts access.Facts
	if err := json.NewDecoder(response.Body).Decode(&facts); err != nil {
		return access.Facts{}, err
	}
	if facts.Subject != identity.Subject || facts.Issuer != identity.Issuer || facts.Tenant != identity.Tenant || !facts.ValidUntil.After(time.Now()) {
		return access.Facts{}, access.ErrDenied
	}
	if identity.ValidUntil.Before(facts.ValidUntil) {
		facts.ValidUntil = identity.ValidUntil
	}
	return facts, nil
}
func fixtureRemoteProvider(t *testing.T, config *ResourceAccessConfig, endpoint string) access.Provider {
	t.Helper()
	encoded, err := os.ReadFile(config.PublicKeyFile)
	if err != nil {
		t.Fatal(err)
	}
	key, err := jwt.ParseRSAPublicKeyFromPEM(encoded)
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := oauth.New(oauth.Config{Issuer: config.Issuer, Audience: config.Audience, Algorithms: []string{"RS256"}, Keyfunc: func(*jwt.Token) (any, error) { return key, nil }})
	if err != nil {
		t.Fatal(err)
	}
	return fixtureRemoteFacts{verifier: verifier, endpoint: endpoint}
}
