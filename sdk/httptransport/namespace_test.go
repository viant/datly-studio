package httptransport

import (
	"context"
	"github.com/viant/datly-studio/sdk"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestGatewayRequiredNamespaceBlocksBeforeTransport(t *testing.T) {
	for _, operation := range []string{sdk.OperationComponentList, "versions.get", "access.get", "access.replace", "acl.list", "runtime.status", "authorization_predicates.list", "connectors.list", "namespaces.list", "authorization_predicates.types"} {
		for _, id := range []string{"", "invalid", strings.Repeat("a", 64)} {
			calls := 0
			gateway := Gateway{Config: Config{Mode: Development, DevelopmentSubject: "owner", RequireNamespace: true}, Transport: transportFunc(func(context.Context, string, any, any) error { calls++; return nil })}
			req := httptest.NewRequest(http.MethodPost, PathPrefix+operation, nil)
			req.RemoteAddr = "127.0.0.1:1234"
			req.Header.Set("X-Studio-Development-Subject", "owner")
			if id != "" {
				req.Header.Set(NamespaceHeader, id)
			}
			res := httptest.NewRecorder()
			gateway.ServeHTTP(res, req)
			required := sdk.RequiresNamespaceSelection(operation)
			blocked := (required && id == "") || id == "invalid"
			if blocked && (res.Code != 400 || calls != 0) {
				t.Fatalf("%s/%q unscoped call reached transport: %d calls=%d", operation, id, res.Code, calls)
			}
			if !blocked && (res.Code != 200 || calls != 1) {
				t.Fatalf("%s/%q valid/global request denied: %d calls=%d body=%s", operation, id, res.Code, calls, res.Body.String())
			}
		}
	}
}
