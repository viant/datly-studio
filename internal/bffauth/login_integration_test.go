package bffauth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"sync"
	"testing"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/viant/datly-studio/sdk/access"
	accessoauth "github.com/viant/datly-studio/sdk/access/oauth"
	"github.com/viant/datly-studio/sdk/httptransport"
	accessstore "github.com/viant/datly-studio/store/sql/access"
	"github.com/viant/datly-studio/store/sql/migrate"
	"github.com/viant/scy/auth/jwt/verifier"
	"golang.org/x/oauth2"
	_ "modernc.org/sqlite"
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
	issueSubject := "alice"
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
			selectedSubject := issueSubject
			lock.Unlock()
			sum := sha256.Sum256([]byte(verifierValue))
			if !found || base64.RawURLEncoding.EncodeToString(sum[:]) != challenge {
				http.Error(w, "PKCE mismatch or replay", http.StatusBadRequest)
				return
			}
			claims := accessoauth.Claims{RegisteredClaims: jwtv5.RegisteredClaims{Issuer: selectedIssuer, Audience: jwtv5.ClaimStrings{selectedAudience}, Subject: selectedSubject, ExpiresAt: jwtv5.NewNumericDate(time.Now().Add(time.Hour))}, Tenant: "one", Roles: []string{"reviewer"}}
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
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "policies.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	migrations, err := migrate.New()
	if err != nil {
		t.Fatal(err)
	}
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	policyStore := &accessstore.Store{DB: db}
	defer policyStore.Close(context.Background())
	resource := access.Resource{Kind: "report", ID: "operations", Tenant: "one", Version: "3"}
	view := access.Policy{Mode: "protected", Rule: &access.Rule{Kind: "subject", Value: "alice"}}
	manage := access.Policy{Mode: "protected", Rule: &access.Rule{Kind: "role", Value: "access-admin"}}
	if _, err := policyStore.Provision(context.Background(), access.Document{Resource: resource, Policies: map[string]access.Policy{
		"viewAccess": view, "manageAccess": manage, "preview": {Mode: "protected", Rule: &access.Rule{Kind: "role", Value: "reviewer"}},
	}}, "bootstrap"); err != nil {
		t.Fatal(err)
	}
	accessProvider, err := accessoauth.New(accessoauth.Config{Issuer: issuer, Audience: audience, Algorithms: []string{"RS256"}, Keyfunc: func(*jwtv5.Token) (any, error) { return &privateKey.PublicKey, nil }})
	if err != nil {
		t.Fatal(err)
	}
	mux.Handle("POST "+httptransport.PathPrefix, httptransport.Gateway{Config: httptransport.Config{Mode: httptransport.Authenticated, Authenticator: sessions},
		Transport: &access.Transport{Service: &access.Service{Store: policyStore, Provider: accessProvider}}})
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
	callAccess := func(client *http.Client, operation string, body any, forgedBearer string) (*http.Response, error) {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		request, err := http.NewRequest(http.MethodPost, studio.URL+httptransport.PathPrefix+operation, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		request.Header.Set("Content-Type", "application/json")
		if forgedBearer != "" {
			request.Header.Set("Authorization", "Bearer "+forgedBearer)
		}
		return client.Do(request)
	}
	for _, operation := range []string{access.OperationGet, access.OperationContext} {
		answer, err := callAccess(browser, operation, resource, "browser-forgery")
		if err != nil {
			t.Fatal(err)
		}
		if answer.StatusCode != http.StatusOK {
			answer.Body.Close()
			t.Fatalf("verified BFF %s status=%d", operation, answer.StatusCode)
		}
		if operation == access.OperationGet {
			var policy access.Document
			if err := json.NewDecoder(answer.Body).Decode(&policy); err != nil || policy.Resource != resource || policy.Revision != 1 {
				t.Fatalf("verified policy=%+v err=%v", policy, err)
			}
		} else {
			var context access.EditorContext
			if err := json.NewDecoder(answer.Body).Decode(&context); err != nil || context.CanManage || context.Source != "verified-principal" || len(context.Choices.Roles) != 1 || context.Choices.Roles[0].ID != "reviewer" {
				t.Fatalf("verified editor context=%+v err=%v", context, err)
			}
		}
		answer.Body.Close()
	}
	writeAttempt := access.Document{Resource: resource, Revision: 1, Policies: map[string]access.Policy{
		"viewAccess": view, "manageAccess": manage, "preview": {Mode: "public"},
	}}
	blockedWrite, err := callAccess(browser, access.OperationReplace, writeAttempt, "browser-forgery")
	if err != nil {
		t.Fatal(err)
	}
	blockedWrite.Body.Close()
	if blockedWrite.StatusCode != http.StatusForbidden {
		t.Fatalf("read-only BFF policy replacement status=%d", blockedWrite.StatusCode)
	}
	unchanged, err := policyStore.Get(context.Background(), resource)
	if err != nil || unchanged.Revision != 1 || unchanged.Policies["preview"].Mode != "protected" {
		t.Fatalf("read-only BFF changed policy=%+v err=%v", unchanged, err)
	}
	unauthenticated := newBrowser()
	denied, err := callAccess(unauthenticated, access.OperationGet, resource, "browser-forgery")
	if err != nil {
		t.Fatal(err)
	}
	denied.Body.Close()
	if denied.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bearer without BFF cookie status=%d", denied.StatusCode)
	}
	lock.Lock()
	issueSubject = "bob"
	lock.Unlock()
	bobBrowser := newBrowser()
	bobLogin, err := bobBrowser.Get(studio.URL + loginPath)
	if err != nil {
		t.Fatal(err)
	}
	bobLogin.Body.Close()
	if bobLogin.StatusCode != http.StatusOK {
		t.Fatalf("bob login status=%d", bobLogin.StatusCode)
	}
	for _, operation := range []string{access.OperationGet, access.OperationContext} {
		answer, err := callAccess(bobBrowser, operation, resource, "")
		if err != nil {
			t.Fatal(err)
		}
		answer.Body.Close()
		if answer.StatusCode != http.StatusForbidden {
			t.Fatalf("other signed principal %s status=%d", operation, answer.StatusCode)
		}
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
