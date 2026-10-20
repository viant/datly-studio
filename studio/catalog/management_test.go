package catalog

import (
	"context"
	"encoding/json"
	"github.com/viant/datly-studio/runtime/managementwindow"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	namespaceext "github.com/viant/mcp-protocol/extension/namespaces"
	"github.com/viant/scy/auth/jwt"
	"strings"
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
	if err != nil || ns.Name != "operations" || len(definitions) == 0 {
		t.Fatalf("authenticated discovery failed: namespace=%+v definitions=%d error=%v", ns, len(definitions), err)
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
	var connection map[string]any
	for _, definition := range defs {
		if definition.URI == "mcp://datastudio/admin" {
			if err := json.Unmarshal(definition.JSON, &connection); err != nil {
				t.Fatal(err)
			}
		}
		if strings.HasPrefix(definition.URI, "datasource://datastudio/") {
			var resource struct {
				DataSource struct {
					Backend struct{ Kind, Service string }
				}
			}
			if err := json.Unmarshal(definition.JSON, &resource); err != nil {
				t.Fatal(err)
			}
			if resource.DataSource.Backend.Kind == "mcp_tool" && resource.DataSource.Backend.Service != "mcp://datastudio/admin" {
				t.Fatal("management datasource lost canonical MCP binding")
			}
		}
	}
	if connection == nil || connection["transport"].(map[string]any)["url"] != "https://studio.example/admin/v1/studio/sdk-mcp/mcp" || connection["primitiveProviderIdentity"] != nil {
		t.Fatalf("invalid tool-only administrative connection: %v", connection)
	}
	for _, definition := range defs {
		if strings.HasPrefix(definition.URI, "navigation://") {
			var nav map[string]any
			if err := json.Unmarshal(definition.JSON, &nav); err != nil {
				t.Fatal(err)
			}
			if nav["schemaVersion"] != float64(1) {
				t.Fatal("navigation envelope metadata lost")
			}
			var check func([]any)
			seenWindows := map[string]bool{}
			check = func(menus []any) {
				for _, raw := range menus {
					item := raw.(map[string]any)
					if children, ok := item["children"].([]any); ok {
						check(children)
						continue
					}
					if action, ok := item["action"].(map[string]any); ok && action["type"] == "url" {
						t.Fatal("external native action remains")
					}
				if action, ok := item["action"].(map[string]any); ok && action["type"] == "window" {
					uri, _ := action["resourceUri"].(string)
					if !strings.HasPrefix(uri, "window://datastudio/") {
						t.Fatal("management leaf must use its own namespace")
					}
					seenWindows[uri] = true
				}
				}
			}
			for _, raw := range nav["applications"].([]any) {
				app := raw.(map[string]any)
				check(app["menus"].([]any))
				if len(app["styleResources"].([]any)) != 1 {
					t.Fatal("native stylesheet not bound")
				}
			}
			for _, uri := range []string{"window://datastudio/schema", "window://datastudio/runtime"} {
				if !seenWindows[uri] {
					t.Fatalf("functional management window absent from navigation: %s", uri)
				}
			}
		}
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

func TestBinaryManagementResourcesAreCopiedScopedAndAuthenticated(t *testing.T) {
	managementSettings.Lock()
	priorConfig, priorResources := managementSettings.config, managementSettings.resources
	managementSettings.config = nil
	managementSettings.resources = nil
	managementSettings.Unlock()
	t.Cleanup(func() {
		managementSettings.Lock()
		managementSettings.config = priorConfig
		managementSettings.resources = priorResources
		managementSettings.Unlock()
	})
	t.Setenv("STUDIO_MANAGEMENT_NAMESPACE", "operations")
	t.Setenv("STUDIO_MANAGEMENT_PROVIDER_ID", "provider")
	t.Setenv("STUDIO_MANAGEMENT_MCP_SERVICE", "management")
	t.Setenv("STUDIO_MANAGEMENT_PUBLIC_URL", "https://studio.example")
	t.Setenv("STUDIO_MANAGEMENT_SECTIONS_JSON", "")
	own := []byte(`{"schemaVersion":1,"css":".workspace { color: inherit; }"}`)
	definitions := []managementwindow.Definition{{URI: "style://operations/workspace", JSON: own}, {URI: "style://foreign/workspace", JSON: []byte(`{"schemaVersion":1}`)}}
	if err := ConfigureManagementResources(definitions); err != nil {
		t.Fatal(err)
	}
	own[0] = 0
	claims := &jwt.Claims{}
	claims.Subject = "alice"
	auth := &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "alice"}}
	_, resources, err := management(claims, auth)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, resource := range resources {
		if resource.URI == "style://foreign/workspace" {
			t.Fatal("foreign registration escaped namespace scope")
		}
		if resource.URI == "style://operations/workspace" {
			found = true
			if !json.Valid(resource.JSON) {
				t.Fatal("caller mutated stored registration")
			}
			resource.JSON[0] = 0
		}
	}
	if !found {
		t.Fatal("registered style missing")
	}
	output := &ResourceOutput{}
	if err := runResource(context.Background(), nil, &ResourceInput{Jwt: claims, Auth: auth, Request: namespaceext.ResourceGetRequest{URI: "style://operations/workspace"}}, output); err != nil || !json.Valid(output.Response) {
		t.Fatalf("registered resource read: %v", err)
	}
	if _, _, err := management(nil, auth); err == nil {
		t.Fatal("anonymous registration discovery allowed")
	}
	for _, invalid := range [][]managementwindow.Definition{{{URI: "https://example.com", JSON: []byte(`{}`)}}, {{URI: "skill://operations/item", JSON: []byte(`{}`)}}, {{URI: "style://operations/item", JSON: []byte(`invalid`)}}} {
		if err := ConfigureManagementResources(invalid); err == nil {
			t.Fatal("invalid registration accepted")
		}
	}
}
