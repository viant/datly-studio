package sdk

import (
	"context"
	"github.com/viant/authz"
	"github.com/viant/datly-studio/runtime/accessprovider"
	"testing"
	"time"
)

type namespaceAudienceFacts struct{ facts authz.Facts }

func (p namespaceAudienceFacts) Resolve(context.Context) (authz.Facts, error) { return p.facts, nil }

func TestNamespaceAudienceMatchesNativeVisibilityWithoutGrantingManagement(t *testing.T) {
	facts := authz.Facts{Subject: "viewer", Issuer: "https://idp.example.test", Tenant: "example", Roles: []string{"READER"}, ValidUntil: time.Now().Add(time.Minute)}
	ctx := accessprovider.WithProvider(context.Background(), namespaceAudienceFacts{facts})
	namespace := Namespace{NamespaceID: "stored-id", OwnerID: "owner", Status: "active", Visibility: "private", AllowedRoles: []string{"READER"}}
	if !CanViewNamespace(ctx, "viewer", namespace) || CanManageNamespace(ctx, "viewer", namespace) {
		t.Fatal("viewer audience must not confer management")
	}
	namespace.AllowedRoles = nil
	if CanViewNamespace(ctx, "viewer", namespace) {
		t.Fatal("unassigned role viewed private namespace")
	}
	namespace.ManagerRoles = []string{"READER"}
	if !CanViewNamespace(ctx, "viewer", namespace) || !CanManageNamespace(ctx, "viewer", namespace) {
		t.Fatal("manager lost namespace audience")
	}
	if CanViewNamespace(ctx, "other", namespace) {
		t.Fatal("caller subject differed from verified facts")
	}
	namespace.Status = "archived"
	if CanViewNamespace(ctx, "owner", namespace) {
		t.Fatal("archived namespace admitted owner")
	}
	namespace.Status = "active"
	namespace.ManagerRoles = nil
	namespace.Visibility = "public"
	if !CanViewNamespace(ctx, "viewer", namespace) {
		t.Fatal("authenticated public namespace hidden")
	}
	namespace.Visibility = "private"
	namespace.AllowedRoles = []string{"READER"}
	facts.ValidUntil = time.Now().Add(-time.Second)
	if CanViewNamespace(accessprovider.WithProvider(context.Background(), namespaceAudienceFacts{facts}), "viewer", namespace) {
		t.Fatal("expired role facts admitted")
	}
}
