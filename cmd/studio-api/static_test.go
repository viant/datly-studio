package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/viant/datly-studio/internal/bffauth"
	"github.com/viant/datly-studio/sdk/httptransport"
)

func TestStaticAssetsServeOnlyPublicFilesUnderRoot(t *testing.T) {
	directory := t.TempDir()
	for name, body := range map[string]string{
		"index.html": "<main>Studio</main>",
		"app.js":     "export default 1",
		".env":       "private",
	} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(directory, "assets"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "assets", "app.css"), []byte("body{}"), 0600); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(directory, "outside.txt")); err != nil {
		t.Fatal(err)
	}
	assets, err := newStaticAssets(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer assets.Close()
	for _, test := range []struct {
		method, path string
		status       int
		body         string
	}{
		{http.MethodGet, "/", http.StatusOK, "<main>Studio</main>"},
		{http.MethodGet, "/app.js", http.StatusOK, "export default 1"},
		{http.MethodHead, "/assets/app.css", http.StatusOK, ""},
		{http.MethodGet, "/assets", http.StatusNotFound, ""},
		{http.MethodGet, "/.env", http.StatusNotFound, ""},
		{http.MethodGet, "/outside.txt", http.StatusNotFound, ""},
		{http.MethodGet, "/assets/../.env", http.StatusNotFound, ""},
		{http.MethodGet, "/assets%2fapp.css", http.StatusNotFound, ""},
		{http.MethodPost, "/app.js", http.StatusMethodNotAllowed, ""},
	} {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			response := httptest.NewRecorder()
			assets.ServeHTTP(response, request)
			if response.Code != test.status || test.body != "" && response.Body.String() != test.body {
				t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
			}
			if test.status == http.StatusOK && response.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatal("static asset is missing nosniff")
			}
			if strings.HasPrefix(test.path, "/.") && strings.Contains(response.Body.String(), "private") {
				t.Fatal("hidden file leaked")
			}
		})
	}
}

func TestStaticAssetsRequireAbsoluteDirectory(t *testing.T) {
	if _, err := newStaticAssets("relative/dist"); err == nil {
		t.Fatal("relative static root accepted")
	}
	if _, err := newStaticAssets(string(os.PathSeparator)); err == nil {
		t.Fatal("filesystem root accepted")
	}
	if _, err := newStaticAssets(t.TempDir()); err == nil {
		t.Fatal("directory without index.html accepted")
	}
	file := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(file, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := newStaticAssets(file); err == nil {
		t.Fatal("regular file accepted as static root")
	}
}

func TestStaticFallbackDoesNotReplaceBFFRoutes(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "index.html"), []byte("public UI"), 0600); err != nil {
		t.Fatal(err)
	}
	assets, err := newStaticAssets(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer assets.Close()
	mux := http.NewServeMux()
	mux.Handle("/", assets)
	mux.HandleFunc("GET /v1/studio/auth/me", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	for _, test := range []struct {
		path   string
		status int
	}{
		{"/", http.StatusOK},
		{"/v1/studio/auth/me", http.StatusUnauthorized},
		{"/v1/studio/auth/unknown", http.StatusNotFound},
	} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
		if response.Code != test.status {
			t.Fatalf("%s: got %d, want %d", test.path, response.Code, test.status)
		}
	}
}

func TestStaticUIAndAuthenticatedExtensionShareOneBFFOrigin(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "index.html"), []byte("public UI"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "app-config.json"), []byte(`{"mode":"authenticated"}`), 0600); err != nil {
		t.Fatal(err)
	}
	assets, err := newStaticAssets(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer assets.Close()
	var forwardedAuth, forwardedCookie string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwardedAuth, forwardedCookie = r.Header.Get("Authorization"), r.Header.Get("Cookie")
		if r.URL.Path != "/api/widgets/run" {
			t.Errorf("backend path=%q", r.URL.Path)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer backend.Close()
	sessions, err := bffauth.New(bffauth.Config{}, extensionVerifier{})
	if err != nil {
		t.Fatal(err)
	}
	proxyConfig, err := resolveExtensionProxyConfig(string(httptransport.Authenticated), backend.URL, "/api/widgets")
	if err != nil {
		t.Fatal(err)
	}
	proxy, err := newExtensionProxy(sessions, proxyConfig)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", assets)
	sessions.Register(mux)
	handler := cors("https://studio.example", true, noStore(routeExtensionProxy(mux, proxy)))
	for _, path := range []string{"/", "/app-config.json"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("public asset %s: status=%d headers=%v", path, response.Code, response.Header())
		}
	}
	login := httptest.NewRequest(http.MethodPost, "/v1/studio/auth/session", nil)
	login.Header.Set("Origin", "https://studio.example")
	login.Header.Set("Authorization", "Bearer verified-user-token")
	loginResponse := httptest.NewRecorder()
	handler.ServeHTTP(loginResponse, login)
	if loginResponse.Code != http.StatusCreated || len(loginResponse.Result().Cookies()) != 1 {
		t.Fatalf("session exchange: %d %s", loginResponse.Code, loginResponse.Body.String())
	}
	request := httptest.NewRequest(http.MethodPost, extensionProxyMount+"/api/widgets/run", strings.NewReader("{}"))
	request.Header.Set("Origin", "https://studio.example")
	request.Header.Set("Authorization", "Bearer attacker-token")
	request.AddCookie(loginResponse.Result().Cookies()[0])
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted || forwardedAuth != "Bearer verified-user-token" || forwardedCookie != "" {
		t.Fatalf("extension: status=%d forwarded auth=%q cookie=%q", response.Code, forwardedAuth, forwardedCookie)
	}
	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, extensionProxyMount+"/api/widgets/run", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized extension status=%d", unauthorized.Code)
	}
	// The same cookie is accepted by the existing session endpoint; the
	// static fallback never handles or overwrites that route.
	me := httptest.NewRequest(http.MethodGet, "/v1/studio/auth/me", nil)
	me.AddCookie(loginResponse.Result().Cookies()[0])
	meResponse := httptest.NewRecorder()
	handler.ServeHTTP(meResponse, me)
	if meResponse.Code != http.StatusOK {
		t.Fatalf("session route status=%d", meResponse.Code)
	}
}
