package studioapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestModuleMountsAreExplicitAndCannotShadowSDK(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	for _, path := range []string{"/", "/mcp", "/v1/studio/sdk-mcp/", "/api/../v1/", "/api/resources?x=1", "/api/{name}", "/api//test"} {
		if mountModules(http.NewServeMux(), []Module{{path, h}}) == nil {
			t.Fatalf("unsafe mount %q", path)
		}
	}
	if mountModules(http.NewServeMux(), []Module{{"/api/resources/mcp", h}, {"/api/resources/mcp", h}}) == nil {
		t.Fatal("duplicate mount")
	}
	mux := http.NewServeMux()
	if err := mountModules(mux, []Module{{"/api/resources/mcp", h}, {"/api/resources/preview/", h}}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/resources/mcp", "/api/resources/preview/window"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("POST", path, nil))
		if w.Code != 204 {
			t.Fatalf("%s = %d", path, w.Code)
		}
	}
}
