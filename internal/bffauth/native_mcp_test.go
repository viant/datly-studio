package bffauth

import (
	"context"
	"fmt"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/internal/accessconfig"
	"github.com/viant/datly-studio/internal/datatest"
	authtransport "github.com/viant/mcp/client/auth/transport"
)

func TestNativeMCPProxyPreservesBearerForNativeVerifierAndCookieSession(t *testing.T) {
	fixture := datatest.NewJWTFixture(t)
	keyFile := filepath.Join(t.TempDir(), "public.pem")
	if err := os.WriteFile(keyFile, fixture.PublicKeyPEM(t), 0600); err != nil {
		t.Fatal(err)
	}
	verifier, err := accessconfig.NewClaimsVerifier(accessconfig.Config{PublicKeyFile: keyFile})
	if err != nil {
		t.Fatal(err)
	}
	token := strings.TrimPrefix(fixture.BearerWithClaims(t, jwtlib.MapClaims{"iss": "https://issuer.example", "sub": "owner", "exp": time.Now().Add(time.Minute).Unix()}), "Bearer ")
	verified := 0
	native := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" || r.Header.Get("Cookie") != "" || r.Header.Get("X-Studio-Runtime-Token") != "" {
			t.Error("unexpected native request boundary")
		}
		bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		claims, err := verifier.Verify(oauth.WithBearer(r.Context(), bearer))
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if claims["sub"] != "owner" {
			t.Error("native verified another subject")
		}
		verified++
		// The upstream cannot consume browser cookies. Its freshly minted
		// transport cookie must not rotate the caller's metadata identity.
		http.SetCookie(w, &http.Cookie{Name: "BFF-Auth-Session", Value: "upstream-session", Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: DefaultCookieName, Value: "upstream-must-not-replace-browser-session", Path: "/"})
		w.WriteHeader(http.StatusNoContent)
	}))
	defer native.Close()
	target, _ := url.Parse(native.URL)
	service, _ := New(Config{}, verifierStub{})
	id, _, _, err := service.Exchange(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	proxy, err := service.ProxyNativeMCP(target, "/v1/studio/sdk-mcp")
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct {
		auth   string
		cookie bool
		status int
	}{{"Bearer " + token, false, 204}, {"Bearer invalid", true, 401}, {"Basic invalid", true, 401}, {"", true, 204}, {"", false, 401}} {
		request := httptest.NewRequest(http.MethodPost, "/v1/studio/sdk-mcp/mcp", nil)
		if check.auth != "" {
			request.Header.Set("Authorization", check.auth)
		}
		if check.cookie {
			request.AddCookie(&http.Cookie{Name: DefaultCookieName, Value: id})
		}
		request.Header.Set("X-Studio-Runtime-Token", "untrusted")
		response := httptest.NewRecorder()
		proxy.ServeHTTP(response, request)
		if response.Code != check.status {
			t.Fatalf("auth=%t cookie=%t status=%d", check.auth != "", check.cookie, response.Code)
		}
		if len(response.Result().Cookies()) != 0 {
			t.Fatal("upstream cookies escaped the bearer proxy boundary")
		}
	}
	if verified != 2 {
		t.Fatalf("native verified requests=%d", verified)
	}
	blocked := httptest.NewRecorder()
	proxy.ServeHTTP(blocked, httptest.NewRequest(http.MethodPost, "/v1/studio/sdk-mcp/_studio/reload", nil))
	if blocked.Code != 404 {
		t.Fatal("native administrative path exposed")
	}
}

func TestNativeMCPProxyKeepsCredentialScopeStableAcrossReads(t *testing.T) {
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer jwt-token" || r.Header.Get("Cookie") != "" {
			t.Error("upstream did not receive the isolated bearer identity")
		}
		http.SetCookie(w, &http.Cookie{Name: "BFF-Auth-Session", Value: fmt.Sprintf("upstream-%d", calls), Path: "/"})
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":{"skills":[]}}`)
	}))
	defer upstream.Close()
	service, err := New(Config{}, verifierStub{})
	if err != nil {
		t.Fatal(err)
	}
	id, _, _, err := service.Exchange(context.Background(), "jwt-token")
	if err != nil {
		t.Fatal(err)
	}
	target, _ := url.Parse(upstream.URL)
	proxy, err := service.ProxyNativeMCP(target, "/v1/studio/sdk-mcp")
	if err != nil {
		t.Fatal(err)
	}
	front := httptest.NewServer(proxy)
	defer front.Close()
	endpoint, _ := url.Parse(front.URL + "/v1/studio/sdk-mcp/mcp")
	jar, _ := cookiejar.New(nil)
	jar.SetCookies(endpoint, []*http.Cookie{{Name: DefaultCookieName, Value: id, Path: "/"}})
	witness, err := authtransport.New(authtransport.WithCookieJar(jar))
	if err != nil {
		t.Fatal(err)
	}
	before, ok := witness.CredentialScope(endpoint)
	if !ok {
		t.Fatal("credential scope unavailable")
	}
	client := &http.Client{Jar: jar}
	for i := 0; i < 4; i++ {
		request, _ := http.NewRequest(http.MethodPost, endpoint.String(), strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"skills/list","params":{}}`))
		request.Header.Set("Content-Type", "application/json")
		if i%2 == 0 {
			request.Header.Set("Authorization", "Bearer jwt-token")
		}
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil || response.StatusCode != http.StatusOK || string(body) != `{"jsonrpc":"2.0","id":1,"result":{"skills":[]}}` {
			t.Fatalf("read %d: response changed, status=%d error=%v", i, response.StatusCode, err)
		}
		after, valid := witness.CredentialScope(endpoint)
		if !valid || after != before {
			t.Fatalf("read %d invalidated the unchanged caller credential scope", i)
		}
	}
	if calls != 4 {
		t.Fatalf("upstream calls=%d, want 4", calls)
	}
}
