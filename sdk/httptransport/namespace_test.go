package httptransport

import (
	"context"
	"github.com/viant/datly-studio/sdk"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGatewayCarriesNamespaceSelectionAfterAuthentication(t *testing.T) {
	for _, tt := range []struct {
		name    string
		headers []string
		want    int
	}{{"selected", []string{"namespace-id"}, 200}, {"empty", []string{""}, 400}, {"multiple", []string{"a", "b"}, 400}} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			gateway := Gateway{Config: Config{Mode: Authenticated, Authenticator: AuthenticatorFunc(func(ctx context.Context, _ *http.Request) (context.Context, error) {
				return sdk.WithPrincipal(ctx, sdk.Principal{Subject: "owner"}), nil
			})}, Transport: transportFunc(func(ctx context.Context, _ string, _ any, _ any) error {
				calls++
				id, ok := sdk.NamespaceSelectionFromContext(ctx)
				if !ok || id != "namespace-id" {
					t.Fatalf("namespace selection=%q present=%v", id, ok)
				}
				return nil
			})}
			req := httptest.NewRequest(http.MethodPost, PathPrefix+sdk.OperationComponentList, nil)
			for _, value := range tt.headers {
				req.Header.Add(NamespaceHeader, value)
			}
			response := httptest.NewRecorder()
			gateway.ServeHTTP(response, req)
			if response.Code != tt.want {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if tt.want != 200 && calls != 0 {
				t.Fatal("invalid selection reached transport")
			}
		})
	}
}
