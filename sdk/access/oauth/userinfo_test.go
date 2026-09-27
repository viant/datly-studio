package oauth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestUserInfoProviderVerifiesIdentityAndSeparatesFacts(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	responseBody := `{"status":"ok","info":{"uid":"alice","userId":7,"accountId":21,"roles":["writer","reader"],"features":["export"]}}`
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Authorization") == "" || r.Header.Get("Accept") != "application/json" {
			t.Errorf("user-info request headers=%v", r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseBody))
	}))
	defer server.Close()
	provider, err := NewUserInfo(UserInfoConfig{Issuer: "https://identity.example", Audience: "studio-web", Algorithms: []string{"EdDSA"}, Keyfunc: func(*jwt.Token) (any, error) { return public, nil }, URL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	sign := func(change func(*userInfoClaims)) string {
		t.Helper()
		claims := &userInfoClaims{RegisteredClaims: jwt.RegisteredClaims{Issuer: "https://identity.example", Audience: []string{"studio-web"}, Subject: "alice", ExpiresAt: jwt.NewNumericDate(time.Now().Add(59 * time.Minute))}, UserID: 7, AccountID: 21}
		if change != nil {
			change(claims)
		}
		token, signErr := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims).SignedString(private)
		if signErr != nil {
			t.Fatal(signErr)
		}
		return token
	}
	valid := sign(nil)
	facts, err := provider.Resolve(WithBearer(context.Background(), valid))
	if err != nil || facts.Subject != "alice" || facts.Tenant != "21" || facts.Issuer != "https://identity.example" || !facts.ValidUntil.After(time.Now()) || len(facts.Roles) != 2 || facts.Roles[0] != "reader" || facts.Roles[1] != "writer" || len(facts.Exposures) != 1 || facts.Exposures[0] != "export" || len(facts.Entities) != 0 {
		t.Fatalf("facts=%+v error=%v", facts, err)
	}
	if requests != 1 {
		t.Fatalf("user-info calls=%d", requests)
	}
	for name, token := range map[string]string{
		"wrong issuer":      sign(func(c *userInfoClaims) { c.Issuer = "https://other.example" }),
		"wrong audience":    sign(func(c *userInfoClaims) { c.Audience = []string{"other"} }),
		"missing subject":   sign(func(c *userInfoClaims) { c.Subject = "" }),
		"overlong lifetime": sign(func(c *userInfoClaims) { c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(2 * time.Hour)) }),
		"other account":     sign(func(c *userInfoClaims) { c.AccountID = 22 }),
		"other user":        sign(func(c *userInfoClaims) { c.UserID = 8 }),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := provider.Resolve(WithBearer(context.Background(), token)); err == nil {
				t.Fatal("credential was accepted")
			}
		})
	}
	for name, body := range map[string]string{
		"missing roles":            `{"status":"ok","info":{"userId":7,"accountId":21,"features":[]}}`,
		"null features":            `{"status":"ok","info":{"userId":7,"accountId":21,"roles":[],"features":null}}`,
		"different account":        `{"status":"ok","info":{"userId":7,"accountId":22,"roles":[],"features":[]}}`,
		"duplicate role authority": `{"status":"ok","info":{"userId":7,"accountId":21,"roles":[],"roles":["admin"],"features":[]}}`,
		"case-folded duplicate":    `{"status":"ok","info":{"userId":7,"accountId":21,"roles":[],"Roles":["admin"],"features":[]}}`,
		"duplicate grant":          `{"status":"ok","info":{"userId":7,"accountId":21,"roles":["admin","admin"],"features":[]}}`,
	} {
		t.Run(name, func(t *testing.T) {
			responseBody = body
			if _, err := provider.Resolve(WithBearer(context.Background(), valid)); err == nil {
				t.Fatal("malformed user-info authority was accepted")
			}
		})
	}
}

func TestUserInfoProviderRejectsRedirectAndUnsafeEndpoint(t *testing.T) {
	public, private, _ := ed25519.GenerateKey(rand.Reader)
	other := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("bearer was redirected") }))
	defer other.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, other.URL, http.StatusFound) }))
	defer server.Close()
	config := UserInfoConfig{Issuer: "issuer", Audience: "audience", Algorithms: []string{"EdDSA"}, Keyfunc: func(*jwt.Token) (any, error) { return public, nil }, URL: server.URL}
	provider, err := NewUserInfo(config)
	if err != nil {
		t.Fatal(err)
	}
	claims := userInfoClaims{RegisteredClaims: jwt.RegisteredClaims{Issuer: "issuer", Audience: []string{"audience"}, Subject: "alice", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute))}, UserID: 7, AccountID: 21}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims).SignedString(private)
	if _, err := provider.Resolve(WithBearer(context.Background(), token)); err == nil {
		t.Fatal("redirect was accepted")
	}
	config.URL = "http://identity.example/user-info"
	if _, err := NewUserInfo(config); err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("unsafe endpoint error=%v", err)
	}
}
