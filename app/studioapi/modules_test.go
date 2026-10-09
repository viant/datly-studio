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

func TestModuleBearerMCPOriginBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, path, method, origin, authorization string
		status                                    int
		dispatched                                bool
	}{
		{"native MCP", "/api/resources/mcp", "POST", "", "Bearer owned", 204, true},
		{"native delete", "/api/resources/mcp", "DELETE", "", "Bearer owned", 204, true},
		{"explicit foreign origin", "/api/resources/mcp", "POST", "https://foreign.example", "Bearer owned", 403, false},
		{"cookie only", "/api/resources/mcp", "POST", "", "", 403, false},
		{"empty bearer", "/api/resources/mcp", "POST", "", "Bearer", 403, false},
		{"invalid credential", "/api/resources/mcp", "POST", "", "Bearer invalid", 401, true},
		{"SDK still requires origin", "/v1/studio/sdk/components.list", "POST", "", "Bearer owned", 403, false},
		{"unmounted sibling", "/api/resources/mcp-other", "POST", "", "Bearer owned", 403, false},
		{"explicit subtree", "/api/resources/preview/window", "POST", "", "Bearer owned", 204, true},
		{"browser origin", "/api/resources/mcp", "POST", "https://studio.example", "Bearer owned", 204, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dispatched := false
			protected := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				dispatched = true
				if r.Header.Get("Authorization") != "Bearer owned" {
					w.WriteHeader(401)
					return
				}
				w.WriteHeader(204)
			})
			modules := []Module{{"/api/resources/mcp", protected}, {"/api/resources/preview/", protected}}
			mux := http.NewServeMux()
			if err := mountModules(mux, modules); err != nil {
				t.Fatal(err)
			}
			handler := corsModules("https://studio.example", true, modules, mux)
			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Header.Set("Origin", tc.origin)
			req.Header.Set("Authorization", tc.authorization)
			req.Header.Set("Cookie", "studio_session=untrusted")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			if response.Code != tc.status || dispatched != tc.dispatched {
				t.Fatalf("status=%d dispatched=%v", response.Code, dispatched)
			}
		})
	}
}
