package studioapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConfiguredHTTPMiddlewareKeepsStandaloneAndRejectsMissingBoundary(t *testing.T) {
	original := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(201) })
	handler, err := configuredHTTPMiddleware(nil, original)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/", nil))
	if response.Code != 201 {
		t.Fatal("unset middleware changed standalone")
	}
	handler, err = configuredHTTPMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/denied" {
				w.WriteHeader(403)
				return
			}
			next.ServeHTTP(w, r)
		})
	}, original)
	if err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/denied", nil))
	if response.Code != 403 {
		t.Fatal("boundary not applied")
	}
	if _, err = configuredHTTPMiddleware(func(http.Handler) http.Handler { return nil }, original); err == nil {
		t.Fatal("missing configured boundary accepted")
	}
}
