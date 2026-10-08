package host

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	access "github.com/viant/authz"
	"github.com/viant/datly-studio/runtime/accessprovider"
)

type contextResourceProvider struct{}

func (contextResourceProvider) Resolve(context.Context) (access.Facts, error) {
	return access.Facts{}, nil
}

func TestResourceCredentialHTTPSeedsNativeProviderContext(t *testing.T) {
	provider := contextResourceProvider{}
	service := &Service{resourceAccess: &access.Service{Provider: provider}}
	var resolved access.Provider
	handler := service.resourceCredential(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		resolved, err = accessprovider.FromEnvironment(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if response.Code != http.StatusNoContent || resolved != provider {
		t.Fatalf("status=%d resolved=%T", response.Code, resolved)
	}
}
