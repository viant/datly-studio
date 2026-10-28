package studioapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type publicMetadataFixture struct{}

func (publicMetadataFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }
func (publicMetadataFixture) AllowAnonymousRequest(r *http.Request) bool {
	return r.Header.Get("Authorization") == ""
}
func TestPublicMetadataModuleAllowsOriginlessRequestsOnlyOnItsMount(t *testing.T) {
	handler := corsModules("https://studio.example", true, []Module{{Path: "/api/metadata/", Handler: publicMetadataFixture{}}}, publicMetadataFixture{})
	for _, test := range []struct {
		path, origin string
		want         int
	}{{"/api/metadata/team/mcp", "", 204}, {"/v1/studio/write", "", 403}, {"/api/metadata/team/mcp", "https://foreign.example", 403}} {
		req := httptest.NewRequest("POST", test.path, nil)
		if test.origin != "" {
			req.Header.Set("Origin", test.origin)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != test.want {
			t.Fatalf("%s origin %q: got %d want %d", test.path, test.origin, response.Code, test.want)
		}
	}
}
