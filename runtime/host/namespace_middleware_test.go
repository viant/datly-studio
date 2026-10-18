package host

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/viant/datly-studio/runtime/namespacemcp"
)

func TestNamespaceHTTPMiddlewareRetainsEndpointIsolation(t *testing.T) {
	for _, id := range []string{strings.Repeat("a", 64), strings.Repeat("b", 64)} {
		source := &namespaceRuntimeSource{runtime: &Service{config: Config{Authentication: Authentication{DefaultMode: "public"}, NamespaceHTTPMiddleware: func(endpoint namespacemcp.Endpoint, next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Namespace", endpoint.NamespaceID)
				next.ServeHTTP(w, r)
			})
		}}}, endpoint: namespacemcp.Endpoint{NamespaceID: id}}
		next := source.WrapHTTP(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
		result := httptest.NewRecorder()
		next.ServeHTTP(result, httptest.NewRequest(http.MethodPost, "/mcp", nil))
		if result.Code != http.StatusNoContent || result.Header().Get("Namespace") != id {
			t.Fatalf("namespace wrapper: %d %s", result.Code, result.Header().Get("Namespace"))
		}
	}
}

func TestNativeNamespaceHandlerPreservesAuthenticationWithoutComposition(t *testing.T) {
	for _, mode := range []string{"required", "public"} {
		composed := false
		source := &namespaceRuntimeSource{runtime: &Service{config: Config{
			Authentication:          Authentication{DefaultMode: mode},
			NamespaceHTTPMiddleware: func(_ namespacemcp.Endpoint, next http.Handler) http.Handler { composed = true; return next },
		}}}
		handler := source.WrapNativeHTTP(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) }))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("POST", "/mcp", nil))
		want := 204
		if mode == "required" {
			want = 401
		}
		if w.Code != want || composed {
			t.Fatalf("mode=%s status=%d composed=%t", mode, w.Code, composed)
		}
	}
}
