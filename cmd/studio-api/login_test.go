package main

import (
	"strings"
	"testing"

	"github.com/viant/datly-studio/sdk/httptransport"
)

func TestResolveBFFOAuthLoginConfig(t *testing.T) {
	config, err := resolveLoginConfig(string(httptransport.Authenticated), "https://studio.example", "", "", "", "", "", "openid,profile")
	if err != nil || config != nil {
		t.Fatalf("disabled login config=%+v err=%v", config, err)
	}
	config, err = resolveLoginConfig(string(httptransport.Authenticated), "https://studio.example",
		"https://idp.example/authorize", "https://idp.example/token", "studio-web", "secret", "https://studio.example/v1/studio/auth/callback", "openid,profile,openid")
	if err != nil || config == nil || config.ClientID != "studio-web" || config.ClientSecret != "secret" || len(config.Scopes) != 2 {
		t.Fatalf("valid login config=%+v err=%v", config, err)
	}
	for name, values := range map[string]struct{ mode, auth, token, id, redirect, scopes string }{
		"development":           {string(httptransport.Development), "https://idp.example/authorize", "https://idp.example/token", "studio-web", "http://127.0.0.1:5173/v1/studio/auth/callback", "openid"},
		"missing endpoint":      {string(httptransport.Authenticated), "", "https://idp.example/token", "studio-web", "https://studio.example/v1/studio/auth/callback", "openid"},
		"wrong redirect origin": {string(httptransport.Authenticated), "https://idp.example/authorize", "https://idp.example/token", "studio-web", "https://other.example/v1/studio/auth/callback", "openid"},
		"wrong redirect path":   {string(httptransport.Authenticated), "https://idp.example/authorize", "https://idp.example/token", "studio-web", "https://studio.example/other", "openid"},
		"mixed providers":       {string(httptransport.Authenticated), "https://idp.example/authorize", "https://other.example/token", "studio-web", "https://studio.example/v1/studio/auth/callback", "openid"},
		"missing openid":        {string(httptransport.Authenticated), "https://idp.example/authorize", "https://idp.example/token", "studio-web", "https://studio.example/v1/studio/auth/callback", "profile"},
		"empty scope":           {string(httptransport.Authenticated), "https://idp.example/authorize", "https://idp.example/token", "studio-web", "https://studio.example/v1/studio/auth/callback", "openid,,profile"},
		"URL query":             {string(httptransport.Authenticated), "https://idp.example/authorize?x=1", "https://idp.example/token", "studio-web", "https://studio.example/v1/studio/auth/callback", "openid"},
		"relative endpoints":    {string(httptransport.Authenticated), "/authorize", "/token", "studio-web", "https://studio.example/v1/studio/auth/callback", "openid"},
	} {
		t.Run(name, func(t *testing.T) {
			origin := "https://studio.example"
			if values.mode == string(httptransport.Development) {
				origin = "http://127.0.0.1:5173"
			}
			if got, err := resolveLoginConfig(values.mode, origin, values.auth, values.token, values.id, "", values.redirect, values.scopes); err == nil || got != nil {
				t.Fatalf("unsafe login config=%+v err=%v", got, err)
			}
		})
	}
	if _, err := resolveLoginConfig(string(httptransport.Authenticated), "https://studio.example", "", "", "", "secret", "", "openid"); err == nil || !strings.Contains(err.Error(), "requires") {
		t.Fatalf("orphaned client secret error=%v", err)
	}
}
