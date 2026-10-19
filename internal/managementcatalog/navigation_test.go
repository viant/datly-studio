package managementcatalog

import (
	"encoding/json"
	"net/url"
	"testing"
)

func TestManagementNavigation(t *testing.T) {
	config := Config{Namespace: "operations", PublicURL: "https://studio.example/manage?theme=dark"}
	nav, err := navigation(config)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Applications []struct {
			ID, Title, Icon string
			Menus           []struct {
				ID, Title string
				Action    struct{ Type, URL, ResourceURI string }
			}
		}
	}
	if err := json.Unmarshal(nav.JSON, &envelope); err != nil {
		t.Fatal(err)
	}
	app := envelope.Applications[0]
	if nav.URI != "navigation://operations/main" || app.ID != "datastudio" || app.Title != "Data Studio" || app.Icon != "database" || len(app.Menus) != 9 {
		t.Fatalf("incorrect navigation: %s", nav.JSON)
	}
	for i, section := range defaultSections {
		action := app.Menus[i].Action
		target, _ := url.Parse(action.URL)
		if action.Type != "url" || target.Host != "studio.example" || target.Query().Get("section") != section.ID || target.Query().Get("theme") != "dark" {
			t.Fatalf("incorrect action: %+v", action)
		}
	}
	if app.Menus[8].Action.ResourceURI != "window://operations/namespaces" {
		t.Fatal("missing portable namespace catalog")
	}
	config.ApplicationID = "host-app"
	config.ApplicationName = "Host Management"
	config.ApplicationIcon = "cog"
	config.Sections = []Section{{"host-section", "Host Screen", "cog"}}
	nav, err = navigation(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(nav.JSON, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Applications[0].ID != "host-app" || len(envelope.Applications[0].Menus) != 2 {
		t.Fatalf("host override lost: %s", nav.JSON)
	}
}

func TestManagementNavigationTrustedURL(t *testing.T) {
	for _, value := range []string{"http://remote.example", "javascript:alert(1)", "https://user:pass@studio.example", "//studio.example", "https://studio.example/#fragment"} {
		if _, err := navigation(Config{Namespace: "operations", PublicURL: value}); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
	for _, value := range []string{"https://studio.example", "http://localhost:8091", "http://127.0.0.1:8091", "http://[::1]:8091"} {
		if _, err := navigation(Config{Namespace: "operations", PublicURL: value}); err != nil {
			t.Fatalf("rejected %q: %v", value, err)
		}
	}
}
