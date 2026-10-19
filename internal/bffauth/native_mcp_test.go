package bffauth

import (
	"context"
	jwtlib "github.com/golang-jwt/jwt/v5"
	"net/http"
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
