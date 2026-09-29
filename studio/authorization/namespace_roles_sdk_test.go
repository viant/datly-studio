package authorization_test

import (
	"context"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/viant/datly-studio/internal/datatest"
	"github.com/viant/datly-studio/sdk"
	sqltransport "github.com/viant/datly-studio/sdk/transport/sql"
	"github.com/viant/datly-studio/studio/authorization"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNamespaceSDKUsesVerifiedRolesAndOwnerManagement(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "namespace_roles_sdk", "studio")
	fixture := datatest.NewJWTFixture(t)
	keyPath := filepath.Join(t.TempDir(), "public.pem")
	if err := os.WriteFile(keyPath, fixture.PublicKeyPEM(t), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STUDIO_ACCESS_ISSUER", "https://namespace.test")
	t.Setenv("STUDIO_ACCESS_AUDIENCE", "studio")
	t.Setenv("STUDIO_ACCESS_PUBLIC_KEY_FILE", keyPath)
	t.Setenv("STUDIO_ACCESS_USER_INFO_URL", "")
	identity := func(subject string, roles []string) context.Context {
		bearer := fixture.BearerWithClaims(t, jwtv5.MapClaims{"sub": subject, "iss": "https://namespace.test", "aud": "studio", "tenant": "test", "exp": time.Now().Add(time.Hour).Unix(), "roles": roles})
		return sdk.WithVerifiedCredential(sdk.WithPrincipal(ctx, sdk.Principal{Subject: subject}), sdk.VerifiedCredential{Bearer: bearer})
	}
	authorizer, err := authorization.NewSDKAuthorizer(db)
	if err != nil {
		t.Fatal(err)
	}
	defer authorizer.Close(ctx)
	client, err := sdk.NewClient(&sqltransport.Transport{DB: db, Authorizer: authorizer})
	if err != nil {
		t.Fatal(err)
	}
	owner := identity("owner", nil)
	analyst := identity("analyst", []string{"forecast_reader"})
	stranger := identity("stranger", nil)
	wrongCase := identity("wrong-case", []string{"FORECAST_READER"})
	created, err := client.Namespaces().Create(owner, sdk.CreateNamespaceInput{Name: "forecasting", Title: "Forecasting", AllowedRoles: []string{"forecast_reader"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, caller := range []context.Context{owner, analyst} {
		got, err := client.Namespaces().Get(caller, "forecasting")
		if err != nil || got.OwnerID != "owner" {
			t.Fatalf("namespace role get: %+v err=%v", got, err)
		}
		page, err := client.Namespaces().List(caller, sdk.ListNamespacesInput{Query: "forecast", Limit: 1})
		if err != nil || len(page.Items) != 1 || page.Items[0].Name != "forecasting" {
			t.Fatalf("namespace role page: %+v err=%v", page, err)
		}
	}
	for _, caller := range []context.Context{stranger, wrongCase} {
		if _, err := client.Namespaces().Get(caller, "forecasting"); err == nil {
			t.Fatal("unassigned role got private namespace")
		}
		page, err := client.Namespaces().List(caller, sdk.ListNamespacesInput{})
		if err != nil || len(page.Items) != 0 {
			t.Fatalf("unassigned role directory: %+v err=%v", page, err)
		}
	}
	title := "Forged title"
	if _, err := client.Namespaces().Update(analyst, "forecasting", sdk.UpdateNamespaceInput{Title: &title, ETag: created.ETag}); err == nil {
		t.Fatal("viewer role modified namespace")
	}
	noRoles := []string{}
	updated, err := client.Namespaces().Update(owner, "forecasting", sdk.UpdateNamespaceInput{AllowedRoles: &noRoles, ETag: created.ETag})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Namespaces().Get(analyst, "forecasting"); err == nil {
		t.Fatal("revoked role still authorized")
	}
	public := "public"
	if _, err = client.Namespaces().Update(owner, "forecasting", sdk.UpdateNamespaceInput{Visibility: &public, ETag: updated.ETag}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Namespaces().Get(stranger, "forecasting"); err != nil {
		t.Fatalf("public namespace denied: %v", err)
	}
	forged := sdk.WithPrincipal(analyst, sdk.Principal{Subject: "owner"})
	if _, err := client.Namespaces().Update(forged, "forecasting", sdk.UpdateNamespaceInput{Title: &title, ETag: updated.ETag + 1}); err == nil {
		t.Fatal("credential subject mismatch managed namespace")
	}

	if _, err := client.Namespaces().Get(forged, "forecasting"); err == nil {
		t.Fatal("credential subject mismatch accepted")
	}
}

func TestNamespaceOwnerRemainsVisibleWhenUserInfoCannotSupplyRoles(t *testing.T) {
	ctx := context.Background()
	db := datatest.OpenSQLite(t, "namespace_owner_userinfo", "studio")
	fixture := datatest.NewJWTFixture(t)
	keyPath := filepath.Join(t.TempDir(), "public.pem")
	if err := os.WriteFile(keyPath, fixture.PublicKeyPEM(t), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STUDIO_ACCESS_ISSUER", "https://namespace.test")
	t.Setenv("STUDIO_ACCESS_AUDIENCE", "studio")
	t.Setenv("STUDIO_ACCESS_PUBLIC_KEY_FILE", keyPath)
	roles := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer roles.Close()
	t.Setenv("STUDIO_ACCESS_USER_INFO_URL", roles.URL)
	identity := func(subject string) context.Context {
		bearer := fixture.BearerWithClaims(t, jwtv5.MapClaims{"sub": subject, "iss": "https://namespace.test", "aud": "studio", "user_id": 7, "account_id": 21, "exp": time.Now().Add(30 * time.Minute).Unix()})
		return sdk.WithVerifiedCredential(sdk.WithPrincipal(ctx, sdk.Principal{Subject: subject}), sdk.VerifiedCredential{Bearer: bearer})
	}
	authorizer, err := authorization.NewSDKAuthorizer(db)
	if err != nil {
		t.Fatal(err)
	}
	defer authorizer.Close(ctx)
	client, err := sdk.NewClient(&sqltransport.Transport{DB: db, Authorizer: authorizer})
	if err != nil {
		t.Fatal(err)
	}
	owner := identity("owner")
	if _, err := client.Namespaces().Create(owner, sdk.CreateNamespaceInput{Name: "forecasting", Title: "Forecasting", AllowedRoles: []string{"forecast_reader"}}); err != nil {
		t.Fatal(err)
	}
	page, err := client.Namespaces().List(owner, sdk.ListNamespacesInput{})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("verified owner lost namespace during user-info outage: %+v err=%v", page, err)
	}
	page, err = client.Namespaces().List(identity("stranger"), sdk.ListNamespacesInput{})
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("role visibility did not fail closed: %+v err=%v", page, err)
	}
	forged := sdk.WithPrincipal(owner, sdk.Principal{Subject: "stranger"})
	if _, err := client.Namespaces().List(forged, sdk.ListNamespacesInput{}); err == nil {
		t.Fatal("mismatched verified subject was accepted")
	}
}
