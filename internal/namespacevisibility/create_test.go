package namespacevisibility

import (
	"context"
	"testing"
	"time"

	"github.com/viant/authz"
	"github.com/viant/authz/oauth"
	"github.com/viant/datly-studio/runtime/accessprovider"
	xexec "github.com/viant/xdatly/exec"
)

func TestNamespaceCreationRequiresConfiguredVerifiedRole(t *testing.T) {
	t.Setenv("STUDIO_NAMESPACE_CREATE_ROLES", "AUTHORING_ADMIN, DATA_ADMIN")
	for _, role := range []string{"AUTHORING_ADMIN", "DATA_ADMIN", "viewer", "authoring_admin", "*", ""} {
		t.Run(role, func(t *testing.T) {
			provider := &adminTestProvider{facts: authz.Facts{Subject: "alice", Issuer: "idp", Tenant: "one", Roles: []string{role}, ValidUntil: time.Now().Add(time.Minute)}}
			ctx := accessprovider.WithProvider(oauth.WithBearer(xexec.WithContext(context.Background(), xexec.New()), "credential"), provider)
			if got, want := CanCreate(ctx, "alice"), role == "AUTHORING_ADMIN" || role == "DATA_ADMIN"; got != want {
				t.Fatalf("CanCreate=%v want=%v", got, want)
			}
			if CanCreate(ctx, "bob") {
				t.Fatal("provider identity mismatch accepted")
			}
		})
	}
}

func TestNamespaceCreationRestrictionUnsetAndInvalidConfiguration(t *testing.T) {
	t.Setenv("STUDIO_NAMESPACE_ADMIN_ROLE", "")
	t.Setenv("STUDIO_NAMESPACE_CREATE_ROLES", "")
	if !CanCreate(context.Background(), "alice") || CanCreate(context.Background(), "") {
		t.Fatal("unconfigured authenticated authoring changed")
	}
	for _, value := range []string{"*", " , "} {
		t.Setenv("STUDIO_NAMESPACE_CREATE_ROLES", value)
		if CanCreate(context.Background(), "alice") {
			t.Fatal("invalid restriction failed open")
		}
	}
}

func TestNamespaceCreationUsesExistingConfiguredAdministratorRole(t *testing.T) {
	t.Setenv("STUDIO_NAMESPACE_CREATE_ROLES", "")
	t.Setenv("STUDIO_NAMESPACE_ADMIN_ROLE", "NAMESPACE_ADMIN")
	provider := &adminTestProvider{facts: authz.Facts{Subject: "alice", Issuer: "idp", Tenant: "one", Roles: []string{"viewer"}, ValidUntil: time.Now().Add(time.Minute)}}
	ctx := accessprovider.WithProvider(oauth.WithBearer(context.Background(), "credential"), provider)
	if CanCreate(ctx, "alice") {
		t.Fatal("configured admin role did not restrict creation")
	}
	provider.facts.Roles = []string{"NAMESPACE_ADMIN"}
	if !CanCreate(ctx, "alice") {
		t.Fatal("configured administrator denied creation")
	}
}

func TestNamespaceSettingsExcludeOwnersAndManagersWithoutAdministratorRole(t *testing.T) {
	t.Setenv("STUDIO_NAMESPACE_CREATE_ROLES", "AUTHORING_ADMIN,DATA_ADMIN")
	grants := `{"manage":["manager"]}`
	for _, role := range []string{"AUTHORING_ADMIN", "DATA_ADMIN", "manager", "viewer", ""} {
		provider := &adminTestProvider{facts: authz.Facts{Subject: "alice", Issuer: "idp", Tenant: "one", Roles: []string{role}, ValidUntil: time.Now().Add(time.Minute)}}
		ctx := accessprovider.WithProvider(oauth.WithBearer(context.Background(), "credential"), provider)
		want := role == "AUTHORING_ADMIN" || role == "DATA_ADMIN"
		for _, owner := range []string{"alice", "someone-else"} {
			if got := CanConfigure(ctx, "alice", owner, &grants); got != want {
				t.Fatalf("role=%q owner=%q got=%v want=%v", role, owner, got, want)
			}
		}
	}
}
