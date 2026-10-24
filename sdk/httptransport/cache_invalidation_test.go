package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/viant/datly-studio/sdk"
)

func TestCacheInvalidationHTTPRequiresVerifiedNamespaceSelection(t *testing.T) {
	called := 0
	gateway := Gateway{Config: Config{Mode: Authenticated, RequireNamespace: true, Authenticator: AuthenticatorFunc(func(ctx context.Context, _ *http.Request) (context.Context, error) {
		return sdk.WithPrincipal(ctx, sdk.Principal{Subject: "verified-editor"}), nil
	})}, Transport: transportFunc(func(ctx context.Context, operation string, input, output any) error {
		called++
		if operation != sdk.OperationVersionInvalidateCache {
			t.Fatalf("operation=%s", operation)
		}
		selection, ok := sdk.NamespaceSelectionFromContext(ctx)
		if !ok || selection != strings.Repeat("a", 64) {
			t.Fatalf("selection=%q", selection)
		}
		*output.(*sdk.CacheViewInvalidation) = sdk.CacheViewInvalidation{ReportID: "reader", VersionNo: 1, View: "rows", Generation: "generation-2"}
		return nil
	})}
	for _, selected := range []string{"", strings.Repeat("a", 64)} {
		request := httptest.NewRequest(http.MethodPost, PathPrefix+sdk.OperationVersionInvalidateCache, strings.NewReader(`{"reportId":"reader","versionNo":1,"routePath":"/reader","view":"rows"}`))
		if selected != "" {
			request.Header.Set(NamespaceHeader, selected)
		}
		response := httptest.NewRecorder()
		gateway.ServeHTTP(response, request)
		if selected == "" && response.Code != http.StatusBadRequest {
			t.Fatalf("unselected status=%d", response.Code)
		}
		if selected != "" {
			var item sdk.CacheViewInvalidation
			if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &item) != nil || item.Generation != "generation-2" {
				t.Fatalf("selected status=%d body=%s", response.Code, response.Body.String())
			}
		}
	}
	if called != 1 {
		t.Fatalf("unselected request reached control plane: calls=%d", called)
	}
}
