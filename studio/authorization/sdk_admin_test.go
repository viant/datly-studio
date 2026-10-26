package authorization

import (
	"context"
	"database/sql"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/internal/namespacevisibility"
	"github.com/viant/datly-studio/runtime/accessprovider"
	"github.com/viant/datly-studio/sdk"
	sqltransport "github.com/viant/datly-studio/sdk/transport/sql"
	"testing"
	"time"
)

type adminFacts struct {
	facts authz.Facts
	calls int
}

func (p *adminFacts) Resolve(context.Context) (authz.Facts, error) { p.calls++; return p.facts, nil }
func TestSDKConfiguredAdministratorNeedsNoPolicyQuery(t *testing.T) {
	t.Setenv("STUDIO_NAMESPACE_ADMIN_ROLE", "NAMESPACE_ADMIN")
	p := &adminFacts{facts: authz.Facts{Subject: "operator", Issuer: "fixture", Tenant: "test", Roles: []string{"NAMESPACE_ADMIN"}, ValidUntil: time.Now().Add(time.Minute)}}
	ctx := accessprovider.WithProvider(sdk.WithPrincipal(context.Background(), sdk.Principal{Subject: "operator"}), p)
	// An unopened database makes any accidental SQL query fail the test.
	a := SDKAuthorizer{DB: &sql.DB{}}
	for _, request := range []sqltransport.AuthorizationRequest{
		{NamespaceName: "other", Permission: "edit"}, {ReportID: "other", Permission: "publish"}, {ConnectorName: "other", Permission: "edit"}, {Permission: "publish"},
	} {
		if err := a.Authorize(ctx, request); err != nil {
			t.Fatal(err)
		}
	}
	if p.calls != 4 {
		t.Fatalf("unexpected identity resolutions: %d", p.calls)
	}
	if err := a.Authorize(ctx, sqltransport.AuthorizationRequest{OwnerID: "someone-else", Permission: "edit"}); err == nil {
		t.Fatal("admin impersonated owner")
	}
	p.facts.Subject = "different"
	// A resource-free request alone has no protected operation, so assert the
	// forged provider identity is rejected by the shared administrator check.
	if namespacevisibility.IsAdministrator(ctx, "operator") {
		t.Fatal("mismatched identity accepted")
	}
}
