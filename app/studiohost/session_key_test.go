package studiohost

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/viant/datly-studio/app/studioapi"
	"github.com/viant/scy"
)

func TestOAuthBrowserSessionKeyStableAndScoped(t *testing.T) {
	ctx := context.Background()
	login := studioapi.Login{ClientID: "test-client", ClientSecret: "synthetic-confidential-client-secret", TokenURL: "https://idp.example/token"}
	origin := "https://aistudio.example"
	key, err := browserSessionKey(ctx, nil, &login, origin)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.StdEncoding.DecodeString(key)
	if err != nil || len(raw) != 32 {
		t.Fatal("invalid derived session key")
	}
	again, err := browserSessionKey(ctx, nil, &login, origin)
	if err != nil || again != key {
		t.Fatal("unchanged OAuth configuration must survive restart")
	}
	for _, field := range []string{"origin", "client", "endpoint", "secret"} {
		t.Run(field, func(t *testing.T) {
			candidate, target := login, origin
			switch field {
			case "origin":
				target = "https://datastudio.example"
			case "client":
				candidate.ClientID = "other-client"
			case "endpoint":
				candidate.TokenURL = "https://other-idp.example/token"
			case "secret":
				candidate.ClientSecret = "rotated-client-secret"
			}
			other, err := browserSessionKey(ctx, nil, &candidate, target)
			if err != nil || other == key {
				t.Fatal("session key must remain isolated to its OAuth application")
			}
		})
	}
	path := filepath.Join(t.TempDir(), "session-key")
	if err := os.WriteFile(path, []byte(key), 0600); err != nil {
		t.Fatal(err)
	}
	override, err := browserSessionKey(ctx, &scy.Resource{URL: path}, nil, "https://other.example")
	if err != nil || override != key {
		t.Fatal("explicit session key override must retain precedence")
	}
}

func TestBrowserSessionKeyWithoutConfidentialClientRemainsRandom(t *testing.T) {
	for _, login := range []*studioapi.Login{nil, {ClientID: "public-client"}} {
		first, err := browserSessionKey(context.Background(), nil, login, "http://127.0.0.1:1234")
		if err != nil {
			t.Fatal(err)
		}
		second, err := browserSessionKey(context.Background(), nil, login, "http://127.0.0.1:1234")
		if err != nil || first == second {
			t.Fatal("public OAuth metadata must never become session encryption material")
		}
	}
}
