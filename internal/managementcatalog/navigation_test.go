package managementcatalog

import (
	"encoding/json"
	"github.com/viant/datly-studio/nativeapp"
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
	var source struct{ Menus []Section }
	if err := json.Unmarshal(nativeapp.Navigation(), &source); err != nil {
		t.Fatal(err)
	}
	for i, section := range source.Menus {
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

func TestEmbeddedNativeNavigationExactMenuParityAndURLActions(t *testing.T) {
	var source map[string]any
	if err := json.Unmarshal(nativeapp.Navigation(), &source); err != nil {
		t.Fatal(err)
	}
	nav, err := navigation(Config{Namespace: "datastudio", PublicURL: "https://studio.example/admin", Navigation: nativeapp.Navigation()})
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct{ Applications []map[string]any }
	if err = json.Unmarshal(nav.JSON, &envelope); err != nil {
		t.Fatal(err)
	}
	var compare func([]any, []any)
	compare = func(want, got []any) {
		if len(want) != len(got) {
			t.Fatalf("menu count changed")
		}
		for i, raw := range want {
			w := raw.(map[string]any)
			g := got[i].(map[string]any)
			for _, key := range []string{"id", "title", "icon"} {
				if w[key] != g[key] {
					t.Fatalf("native menu changed: %s", key)
				}
			}
			if child, ok := w["children"].([]any); ok && len(child) > 0 {
				if _, exists := g["action"]; exists {
					t.Fatal("navigation group must not also have an action")
				}
				compare(child, g["children"].([]any))
				continue
			}
			section, _ := w["section"].(string)
			if section == "" {
				section = w["id"].(string)
			}
			action := g["action"].(map[string]any)
			u, err := url.Parse(action["url"].(string))
			if err != nil || action["type"] != "url" || u.Host != "studio.example" || u.Path != "/admin" || u.Query().Get("section") != section {
				t.Fatalf("incorrect native action: %v", action)
			}
		}
	}
	app := envelope.Applications[0]
	if app["id"] != source["id"] || app["title"] != source["title"] || app["icon"] != source["icon"] {
		t.Fatal("native identity changed")
	}
	compare(source["menus"].([]any), app["menus"].([]any))
}
