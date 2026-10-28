package catalog

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/viant/datly-studio/runtime/managementwindow"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	namespaceext "github.com/viant/mcp-protocol/extension/namespaces"
	"github.com/viant/scy/auth/jwt"
)

func retainManagementSettings(t *testing.T) {
	t.Helper()
	managementSettings.Lock()
	primary, additional, resources := managementSettings.config, managementSettings.additional, managementSettings.resources
	managementSettings.config = nil
	managementSettings.additional = nil
	managementSettings.resources = nil
	managementSettings.Unlock()
	t.Cleanup(func() {
		managementSettings.Lock()
		managementSettings.config = primary
		managementSettings.additional = additional
		managementSettings.resources = resources
		managementSettings.Unlock()
	})
}
func managementApplicationsFixture() []ManagementApplication {
	return []ManagementApplication{
		{Navigation: []byte(`{"id":"operations","title":"Operations","icon":"applications","menus":[{"id":"namespaces","title":"Namespaces"}]}`), PublicURL: "https://studio.example", ProviderIdentity: "operations-management"},
		{PublicURL: "https://studio.example", ProviderIdentity: "datastudio-management"},
	}
}
func TestMultipleManagementApplicationsKeepNamespacesAndResourcesIsolated(t *testing.T) {
	retainManagementSettings(t)
	if err := ConfigureManagementApplications(managementApplicationsFixture()); err != nil {
		t.Fatal(err)
	}
	if err := ConfigureManagementResources([]managementwindow.Definition{
		{URI: "style://operations/workspace", JSON: json.RawMessage(`{"schemaVersion":1,"css":".operations {}"}`)},
		{URI: "style://foreign/workspace", JSON: json.RawMessage(`{"schemaVersion":1,"css":".foreign {}"}`)},
	}); err != nil {
		t.Fatal(err)
	}
	claims := &jwt.Claims{}
	claims.Subject = "alice"
	auth := &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "alice"}}
	catalogs, err := managementCatalogs(claims, auth)
	if err != nil || len(catalogs) != 2 {
		t.Fatalf("catalogs=%d error=%v", len(catalogs), err)
	}
	for _, catalog := range catalogs {
		prefix := "://" + catalog.namespace.Name + "/"
		for _, definition := range catalog.definitions {
			if !strings.Contains(definition.URI, prefix) {
				t.Fatalf("foreign definition escaped namespace: %s", definition.URI)
			}
		}
		out := &GetOutput{}
		if err := runGet(context.Background(), nil, &GetInput{Jwt: claims, Auth: auth, Request: NamespaceGetRequest{URI: catalog.namespace.URI, Kind: "window"}}, out); err != nil {
			t.Fatal(err)
		}
		if len(out.Response.Resources) == 0 {
			t.Fatal("management namespace has no windows")
		}
		for _, ref := range out.Response.Resources {
			if !strings.HasPrefix(ref.URI, "window"+prefix) {
				t.Fatalf("foreign namespace reference: %s", ref.URI)
			}
		}
	}
	for _, uri := range []string{"window://operations/namespaces", "window://datastudio/connectors", "window://datastudio/schema", "style://operations/workspace"} {
		out := &ResourceOutput{}
		if err := runResource(context.Background(), nil, &ResourceInput{Jwt: claims, Auth: auth, Request: namespaceext.ResourceGetRequest{URI: uri}}, out); err != nil || !json.Valid(out.Response) {
			t.Fatalf("read %s: %v", uri, err)
		}
	}
	input := &ResourcesListInput{Jwt: claims, Auth: auth}
	input.Request.Meta.AdditionalProperties = map[string]any{resourceFilterKey: map[string]any{"namespace": "datastudio", "kind": "window", "limit": 100}}
	listed := &ResourcesListOutput{}
	if err := runResourcesList(context.Background(), nil, input, listed); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ref := range listed.Response.Resources {
		if !strings.HasPrefix(ref.Uri, "window://datastudio/") {
			t.Fatal("resource listing crossed namespaces")
		}
		found = found || ref.Uri == "window://datastudio/connectors"
	}
	if !found {
		t.Fatal("secondary catalog connector window not discoverable")
	}
	for _, invalid := range []struct {
		claims *jwt.Claims
		auth   *studioauth.Output
	}{{nil, auth}, {claims, nil}, {claims, &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "bob"}}}} {
		if _, err := managementCatalogs(invalid.claims, invalid.auth); err == nil {
			t.Fatal("unverified catalogs exposed")
		}
		if err := runResource(context.Background(), nil, &ResourceInput{Jwt: invalid.claims, Auth: invalid.auth, Request: namespaceext.ResourceGetRequest{URI: "window://datastudio/connectors"}}, &ResourceOutput{}); err == nil {
			t.Fatal("unverified secondary resource exposed")
		}
	}
}

func TestManagementApplicationConfigurationIsAtomicAndLegacyCallReplacesAll(t *testing.T) {
	retainManagementSettings(t)
	apps := managementApplicationsFixture()
	if err := ConfigureManagementApplications(apps); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range [][]ManagementApplication{nil, {apps[0], apps[0]}, {apps[0], {PublicURL: "https://user:secret@studio.example", ProviderIdentity: "bad"}}} {
		if err := ConfigureManagementApplications(invalid); err == nil {
			t.Fatal("invalid application list accepted")
		}
		managementSettings.RLock()
		count := len(managementSettings.additional)
		managementSettings.RUnlock()
		if count != 1 {
			t.Fatal("failed configuration replaced existing catalogs")
		}
	}
	apps[0].Navigation[0] = 0
	claims := &jwt.Claims{}
	claims.Subject = "alice"
	if _, err := managementCatalogs(claims, &studioauth.Output{Auth: &studioauth.AuthContext{Subject: "alice"}}); err != nil {
		t.Fatal("caller mutated configured navigation:", err)
	}
	if err := ConfigureManagementNavigation(nil, "https://studio.example", "single-provider"); err != nil {
		t.Fatal(err)
	}
	managementSettings.RLock()
	defer managementSettings.RUnlock()
	if len(managementSettings.additional) != 0 || managementSettings.config.Namespace != "datastudio" {
		t.Fatal("legacy single-app configuration retained stale catalogs")
	}
}
