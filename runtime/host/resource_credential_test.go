package host

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/viant/authz/oauth"
	"github.com/viant/mcp-protocol/authorization"
)

func TestResourceCredentialPreservesBearerForDeclaredMCPReaderInputs(t *testing.T) {
	for _, test := range []struct {
		name       string
		headers    []string
		wantStatus int
		wantToken  string
	}{
		{"bearer", []string{"Bearer fixture-credential"}, http.StatusNoContent, "fixture-credential"},
		{"scheme casing", []string{"bearer fixture-credential"}, http.StatusNoContent, "fixture-credential"},
		{"absent", nil, http.StatusNoContent, ""},
		{"duplicate", []string{"Bearer one", "Bearer two"}, http.StatusUnauthorized, ""},
		{"wrong scheme", []string{"Basic one"}, http.StatusUnauthorized, ""},
		{"malformed", []string{"Bearer one two"}, http.StatusUnauthorized, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			called := false
			handler := (&Service{}).resourceCredential(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				if got := oauth.Bearer(r.Context()); got != test.wantToken {
					t.Fatalf("provider credential differs from HTTP source")
				}
				token, _ := r.Context().Value(authorization.TokenKey).(*authorization.Token)
				if test.wantToken == "" {
					if token != nil {
						t.Fatal("missing header created a protocol credential")
					}
				} else if token == nil || token.Token != test.wantToken {
					t.Fatal("declared MCP header inputs lost the transport credential")
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			request := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			for _, h := range test.headers {
				request.Header.Add("Authorization", h)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status=%d want=%d", response.Code, test.wantStatus)
			}
			if called != (test.wantStatus == http.StatusNoContent) {
				t.Fatal("malformed credentials reached the reader")
			}
		})
	}
}
