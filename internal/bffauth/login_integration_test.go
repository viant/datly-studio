package bffauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/viant/scy/auth/jwt/verifier"
	"golang.org/x/oauth2"
)

func TestOAuthLoginWithSignedJWTAndJWKS(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const issuer, audience, keyID = "https://identity.example", "studio-access", "local-test-key"
	var lock sync.Mutex
	challenges := map[string]string{}
	issueIssuer := issuer
	issueAudience := audience
	issueKey := privateKey
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jwks":
			n := base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes())
			e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes())
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{"kty": "RSA", "kid": keyID, "use": "sig", "alg": "RS256", "n": n, "e": e}}})
		case "/authorize":
			query := r.URL.Query()
			if query.Get("code_challenge_method") != "S256" || query.Get("code_challenge") == "" || query.Get("state") == "" {
				http.Error(w, "PKCE is required", http.StatusBadRequest)
				return
			}
			code := "code-" + query.Get("state")
			lock.Lock()
			challenges[code] = query.Get("code_challenge")
			lock.Unlock()
			callback, err := url.Parse(query.Get("redirect_uri"))
			if err != nil {
				http.Error(w, "invalid redirect", http.StatusBadRequest)
				return
			}
			values := callback.Query()
			values.Set("state", query.Get("state"))
			values.Set("code", code)
			callback.RawQuery = values.Encode()
			http.Redirect(w, r, callback.String(), http.StatusFound)
		case "/token":
			if err := r.ParseForm(); err != nil {
				http.Error(w, "invalid form", http.StatusBadRequest)
				return
			}
			code, verifierValue := r.Form.Get("code"), r.Form.Get("code_verifier")
			lock.Lock()
			challenge, found := challenges[code]
			delete(challenges, code)
			selectedIssuer := issueIssuer
			selectedAudience := issueAudience
			selectedKey := issueKey
			lock.Unlock()
			sum := sha256.Sum256([]byte(verifierValue))
			if !found || base64.RawURLEncoding.EncodeToString(sum[:]) != challenge {
				http.Error(w, "PKCE mismatch or replay", http.StatusBadRequest)
				return
			}
			claims := jwtv5.RegisteredClaims{Issuer: selectedIssuer, Audience: jwtv5.ClaimStrings{selectedAudience}, Subject: "alice", ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(time.Hour))}
			token := jwtv5.NewWithClaims(jwtv5.SigningMethodRS256, claims)
			token.Header["kid"] = keyID
			signed, err := token.SignedString(selectedKey)
			if err != nil {
				http.Error(w, "signing failed", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": signed, "token_type": "Bearer", "expires_in": 3600})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()
	mux := http.NewServeMux()
	studio := httptest.NewServer(mux)
	defer studio.Close()
	verified := verifier.New(&verifier.Config{CertURL: provider.URL + "/jwks"})
	if err := verified.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	sessions, err := New(Config{Issuer: issuer, Audience: audience}, verified)
	if err != nil {
		t.Fatal(err)
	}
	login, err := NewLogin(LoginConfig{OAuth: oauth2.Config{
		ClientID: "studio-web", RedirectURL: studio.URL + callbackPath,
		Scopes: []string{"openid", "profile"}, Endpoint: oauth2.Endpoint{AuthURL: provider.URL + "/authorize", TokenURL: provider.URL + "/token"}},
		CookieKey: []byte("0123456789abcdef0123456789abcdef")}, sessions)
	if err != nil {
		t.Fatal(err)
	}
	sessions.Register(mux)
	login.Register(mux)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("Studio")) })
	newBrowser := func() *http.Client {
		jar, err := cookiejar.New(nil)
		if err != nil {
			t.Fatal(err)
		}
		return &http.Client{Jar: jar, Timeout: 5 * time.Second}
	}
	browser := newBrowser()
	response, err := browser.Get(studio.URL + loginPath)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("login status=%d", response.StatusCode)
	}
	me, err := browser.Get(studio.URL + "/v1/studio/auth/me")
	if err != nil {
		t.Fatal(err)
	}
	defer me.Body.Close()
	var identity struct {
		Authenticated bool   `json:"authenticated"`
		Subject       string `json:"subject"`
	}
	if err := json.NewDecoder(me.Body).Decode(&identity); err != nil || me.StatusCode != http.StatusOK || !identity.Authenticated || identity.Subject != "alice" {
		t.Fatalf("verified BFF identity=%+v status=%d err=%v", identity, me.StatusCode, err)
	}
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name     string
		issuer   string
		audience string
		key      *rsa.PrivateKey
	}{
		{"wrong issuer", "https://other.example", audience, privateKey},
		{"wrong audience", issuer, "other-api", privateKey},
		{"wrong signature", issuer, audience, otherKey},
	} {
		t.Run(test.name, func(t *testing.T) {
			lock.Lock()
			issueIssuer, issueAudience, issueKey = test.issuer, test.audience, test.key
			lock.Unlock()
			badBrowser := newBrowser()
			badResponse, err := badBrowser.Get(studio.URL + loginPath)
			if err != nil {
				t.Fatal(err)
			}
			badResponse.Body.Close()
			if badResponse.StatusCode != http.StatusUnauthorized {
				t.Fatalf("rejected login status=%d", badResponse.StatusCode)
			}
			badMe, err := badBrowser.Get(studio.URL + "/v1/studio/auth/me")
			if err != nil {
				t.Fatal(err)
			}
			badMe.Body.Close()
			if badMe.StatusCode != http.StatusUnauthorized {
				t.Fatalf("rejected session status=%d", badMe.StatusCode)
			}
		})
	}
}
