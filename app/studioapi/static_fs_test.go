package studioapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestEmbeddedPublicAssetsRetainRestrictedRouting(t *testing.T) {
	assets, err := newStaticFS(fstest.MapFS{
		"index.html":    &fstest.MapFile{Data: []byte("<main>Embedded Studio</main>")},
		"assets/app.js": &fstest.MapFile{Data: []byte("export default 1")},
		".env":          &fstest.MapFile{Data: []byte("private")},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/", 200}, {"HEAD", "/assets/app.js", 200},
		{"GET", "/assets", 404}, {"GET", "/.env", 404},
		{"GET", "/assets%2fapp.js", 404}, {"GET", "/../.env", 404},
		{"POST", "/assets/app.js", 405},
	} {
		request := httptest.NewRequest(test.method, test.path, nil)
		response := httptest.NewRecorder()
		assets.ServeHTTP(response, request)
		if response.Code != test.status {
			t.Fatalf("%s %s: %d", test.method, test.path, response.Code)
		}
		if response.Code == http.StatusOK && response.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatal("missing asset protection")
		}
	}
	if err = assets.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = newStaticFS(fstest.MapFS{}); err == nil {
		t.Fatal("missing app shell accepted")
	}
}
