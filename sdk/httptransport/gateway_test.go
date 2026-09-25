package httptransport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/viant/datly-studio/sdk"
	scyjwt "github.com/viant/scy/auth/jwt"
)

type transportFunc func(context.Context, string, any, any) error

func (f transportFunc) Invoke(ctx context.Context, operation string, input, output any) error {
	return f(ctx, operation, input, output)
}

func request(t *testing.T, gateway Gateway, operation string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, PathPrefix+operation, nil)
	req.RemoteAddr = "127.0.0.1:8100"
	result := httptest.NewRecorder()
	gateway.ServeHTTP(result, req)
	return result
}

func TestDevelopmentGatewayUsesOnlyConfiguredLoopbackIdentity(t *testing.T) {
	devJWT, err := NewDevelopmentJWT("studio-development", "studio-sdk")
	if err != nil {
		t.Fatal(err)
	}
	transport := transportFunc(func(ctx context.Context, operation string, input, output any) error {
		principal, ok := sdk.PrincipalFromContext(ctx)
		if !ok || principal.Subject != "dev-user" || !principal.Development {
			t.Fatalf("principal=%+v ok=%v", principal, ok)
		}
		credential, ok := sdk.VerifiedCredentialFromContext(ctx)
		claims, claimsOK := credential.Claims.(*scyjwt.Claims)
		if !ok || !claimsOK || claims.Subject != principal.Subject ||
			len(credential.Bearer) < len("Bearer ") || credential.Bearer[:len("Bearer ")] != "Bearer " {
			t.Fatalf("signed development credential missing: ok=%v claims=%+v", ok, claims)
		}
		page := output.(*sdk.ReportPage)
		page.Items = []*sdk.Report{{ID: "report-1", Title: "Revenue"}}
		return nil
	})
	gateway := Gateway{Config: Config{Mode: Development, DevelopmentSubject: "dev-user",
		DevelopmentCredential: devJWT.Credential}, Transport: transport}
	req := httptest.NewRequest(http.MethodPost, PathPrefix+sdk.OperationReportList, nil)
	req.RemoteAddr = "127.0.0.1:8100"
	req.Header.Set("X-Studio-Development-Subject", "dev-user")
	result := httptest.NewRecorder()
	gateway.ServeHTTP(result, req)
	if result.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", result.Code, result.Body.String())
	}
	var page sdk.ReportPage
	if err := json.NewDecoder(result.Body).Decode(&page); err != nil || len(page.Items) != 1 || page.Items[0].ID != "report-1" {
		t.Fatalf("page=%+v err=%v", page, err)
	}
}

func TestDevelopmentGatewayRejectsRemoteOrWrongIdentity(t *testing.T) {
	gateway := Gateway{Config: Config{Mode: Development, DevelopmentSubject: "dev-user"}, Transport: transportFunc(func(context.Context, string, any, any) error { return nil })}
	for _, remote := range []string{"10.0.0.4:8100", "127.0.0.1:8100"} {
		req := httptest.NewRequest(http.MethodPost, PathPrefix+sdk.OperationReportList, nil)
		req.RemoteAddr = remote
		req.Header.Set("X-Studio-Development-Subject", "wrong-user")
		result := httptest.NewRecorder()
		gateway.ServeHTTP(result, req)
		if result.Code != http.StatusUnauthorized {
			t.Fatalf("remote=%s status=%d", remote, result.Code)
		}
	}
}

func TestAuthenticatedGatewayDelegatesVerificationAndRejectsDevelopmentHeader(t *testing.T) {
	called := false
	gateway := Gateway{
		Config: Config{Mode: Authenticated, Authenticator: AuthenticatorFunc(func(ctx context.Context, req *http.Request) (context.Context, error) {
			called = true
			if req.Header.Get("Authorization") != "Bearer signed-token" {
				t.Fatal("bearer was not passed to authenticator")
			}
			return sdk.WithPrincipal(ctx, sdk.Principal{Subject: "user-1"}), nil
		})},
		Transport: transportFunc(func(ctx context.Context, _ string, _ any, output any) error {
			if principal, ok := sdk.PrincipalFromContext(ctx); !ok || principal.Subject != "user-1" {
				t.Fatalf("principal=%+v ok=%v", principal, ok)
			}
			output.(*sdk.ConnectorPage).Items = []*sdk.Connector{{Name: "warehouse"}}
			return nil
		}),
	}
	req := httptest.NewRequest(http.MethodPost, PathPrefix+sdk.OperationConnectorList, nil)
	req.Header.Set("Authorization", "Bearer signed-token")
	result := httptest.NewRecorder()
	gateway.ServeHTTP(result, req)
	if result.Code != http.StatusOK || !called {
		t.Fatalf("status=%d called=%v body=%s", result.Code, called, result.Body.String())
	}
	req.Header.Set("X-Studio-Development-Subject", "dev-user")
	result = httptest.NewRecorder()
	gateway.ServeHTTP(result, req)
	if result.Code != http.StatusUnauthorized {
		t.Fatalf("development header status=%d", result.Code)
	}
}

func TestGatewayRejectsUnknownOperation(t *testing.T) {
	gateway := Gateway{Config: Config{Mode: Development, DevelopmentSubject: "dev-user"}, Transport: transportFunc(func(context.Context, string, any, any) error { return nil })}
	req := httptest.NewRequest(http.MethodPost, PathPrefix+"sql.execute", nil)
	req.RemoteAddr = "127.0.0.1:8100"
	req.Header.Set("X-Studio-Development-Subject", "dev-user")
	result := httptest.NewRecorder()
	gateway.ServeHTTP(result, req)
	if result.Code != http.StatusNotFound {
		t.Fatalf("status=%d", result.Code)
	}
}
