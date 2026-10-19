package catalog

import (
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	"github.com/viant/scy/auth/jwt"
	"testing"
)

func TestManagementRequiresVerifiedPrincipal(t *testing.T) {
	t.Setenv("STUDIO_MANAGEMENT_NAMESPACE", "operations")
	t.Setenv("STUDIO_MANAGEMENT_PROVIDER_ID", "provider")
	t.Setenv("STUDIO_MANAGEMENT_MCP_SERVICE", "management")
	t.Setenv("STUDIO_MANAGEMENT_PUBLIC_URL", "https://studio.example")
	t.Setenv("STUDIO_MANAGEMENT_SECTIONS_JSON", "")
	claims := &jwt.Claims{}
	claims.Subject = "alice"
	auth := &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "alice"}}
	for _, input := range []struct {
		claims *jwt.Claims
		auth   *studioauth.Output
	}{{nil, auth}, {&jwt.Claims{}, auth}, {claims, nil}, {claims, &studioauth.Output{}}, {claims, &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "bob"}}}} {
		if _, _, err := management(input.claims, input.auth); err == nil {
			t.Fatal("unverified principal accepted")
		}
	}
	ns, definitions, err := management(claims, auth)
	if err != nil || ns.Name != "operations" || len(definitions) != 3 {
		t.Fatalf("authenticated discovery failed: %+v %v", ns, err)
	}
	refs := managementReferences(definitions, "navigation")
	if len(refs) != 1 || refs[0].URI != "navigation://operations/main" || refs[0].Name != "Management" {
		t.Fatalf("missing navigation reference: %+v", refs)
	}
}

func TestManagementRejectsInvalidHostNavigation(t *testing.T) {
	t.Setenv("STUDIO_MANAGEMENT_NAMESPACE", "operations")
	t.Setenv("STUDIO_MANAGEMENT_PROVIDER_ID", "provider")
	t.Setenv("STUDIO_MANAGEMENT_MCP_SERVICE", "management")
	claims := &jwt.Claims{}
	claims.Subject = "alice"
	auth := &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "alice"}}
	t.Setenv("STUDIO_MANAGEMENT_SECTIONS_JSON", "malformed")
	if _, _, err := management(claims, auth); err == nil {
		t.Fatal("malformed host sections accepted")
	}
	t.Setenv("STUDIO_MANAGEMENT_SECTIONS_JSON", "")
	t.Setenv("STUDIO_MANAGEMENT_PUBLIC_URL", "http://remote.example")
	if _, _, err := management(claims, auth); err == nil {
		t.Fatal("untrusted host URL accepted")
	}
}

func TestEmbeddedManagementNavigationUsesBinaryIdentityAndVerifiedCatalog(t *testing.T) {
	managementSettings.Lock()
	previous := managementSettings.config
	managementSettings.config = nil
	managementSettings.Unlock()
	t.Cleanup(func() { managementSettings.Lock(); managementSettings.config = previous; managementSettings.Unlock() })
	t.Setenv("STUDIO_MANAGEMENT_NAMESPACE", "caller-config-must-not-select-brand")
	t.Setenv("STUDIO_MANAGEMENT_SECTIONS_JSON", "malformed-old-config")
	if err := ConfigureManagementNavigation(nil, "https://studio.example/admin", "provider"); err != nil {
		t.Fatal(err)
	}
	claims := &jwt.Claims{}
	claims.Subject = "alice"
	auth := &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "alice"}}
	ns, defs, err := management(claims, auth)
	if err != nil || ns.Name != "datastudio" {
		t.Fatalf("embedded namespace unavailable: %v", err)
	}
	refs := managementReferences(defs, "navigation")
	if len(refs) != 1 || refs[0].URI != "navigation://datastudio/main" {
		t.Fatalf("filtered navigation unavailable: %v", refs)
	}
	if _, _, err = management(nil, auth); err == nil {
		t.Fatal("embedded navigation bypassed authentication")
	}
	if err = ConfigureManagementNavigation(nil, "https://user:pass@studio.example", "provider"); err == nil {
		t.Fatal("credential URL accepted")
	}
}
