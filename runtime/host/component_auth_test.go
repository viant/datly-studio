package host

import (
	"context"
	"encoding/json"
	"github.com/viant/scy/auth/jwt"
	"testing"
)

func TestComponentProviderIsolationAndScopes(t *testing.T) {
	var claims jwt.Claims
	if err := json.Unmarshal([]byte(`{"sub":"external-client","iss":"https://runtime.example","aud":["mcp"],"scope":"reports.read"}`), &claims); err != nil {
		t.Fatal(err)
	}
	if !matchesProvider(&claims, OAuthProvider{Issuer: "https://runtime.example", Audience: "mcp"}) {
		t.Fatal("valid runtime identity rejected")
	}
	if matchesProvider(&claims, OAuthProvider{Issuer: "https://studio.example", Audience: "mcp"}) {
		t.Fatal("wrong issuer accepted")
	}
	if matchesProvider(&claims, OAuthProvider{Issuer: "https://runtime.example", Audience: "studio"}) {
		t.Fatal("wrong audience accepted")
	}
	service := &Service{config: Config{Authentication: Authentication{DefaultMode: "public", Components: map[string]ComponentAuthentication{
		"private": {Provider: "runtime", Scopes: []string{"reports.read"}},
		"other":   {Provider: "other"},
		"write":   {Provider: "runtime", Scopes: []string{"reports.write"}},
		"public":  {Public: true},
	}}}}
	ctx := context.WithValue(context.Background(), providerIdentitiesKey{}, map[string]*jwt.Claims{"runtime": &claims})
	if err := service.authorizeRun(ctx, "private"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"other", "write"} {
		if err := service.authorizeRun(ctx, id); err == nil {
			t.Fatalf("unauthorized component %s allowed", id)
		}
	}
	if err := service.authorizeRun(context.Background(), "private"); err == nil {
		t.Fatal("public default bypassed private component policy")
	}
	if err := service.authorizeRun(context.Background(), "public"); err != nil {
		t.Fatal(err)
	}
}
