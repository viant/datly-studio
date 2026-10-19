package managementcatalog

import (
	"encoding/json"
	"fmt"
	"github.com/viant/datly-studio/nativeapp"
	"net"
	"net/url"
	"strings"
)

// Section names are supplied by the host and identify existing management screens.
type Section struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Icon  string `json:"icon"`
}

func navigation(config Config) (Definition, error) {
	if len(config.Navigation) > 0 {
		return embeddedNavigation(config)
	}
	applicationID, title, icon := config.ApplicationID, config.ApplicationName, config.ApplicationIcon
	if applicationID == "" {
		applicationID = "datastudio"
	}
	if title == "" {
		title = "Data Studio"
	}
	if icon == "" {
		icon = "database"
	}
	menus := []any{}
	if config.PublicURL != "" {
		base, err := url.Parse(config.PublicURL)
		if err != nil || base.Host == "" || base.User != nil || base.Opaque != "" || base.Fragment != "" || strings.TrimSpace(config.PublicURL) != config.PublicURL {
			return Definition{}, fmt.Errorf("invalid management public URL")
		}
		host := base.Hostname()
		ip := net.ParseIP(host)
		if base.Scheme != "https" && !(base.Scheme == "http" && (host == "localhost" || ip != nil && ip.IsLoopback())) {
			return Definition{}, fmt.Errorf("management public URL requires HTTPS or loopback HTTP")
		}
		sections := config.Sections
		if sections == nil {
			var app struct {
				Menus []Section `json:"menus"`
			}
			if err := json.Unmarshal(nativeapp.Navigation(), &app); err != nil {
				return Definition{}, err
			}
			sections = app.Menus
		}
		seen := map[string]bool{}
		for _, section := range sections {
			if section.ID == "" || strings.TrimSpace(section.ID) != section.ID || section.Title == "" || seen[section.ID] {
				return Definition{}, fmt.Errorf("invalid management section")
			}
			seen[section.ID] = true
			target := *base
			query := target.Query()
			query.Set("section", section.ID)
			target.RawQuery = query.Encode()
			menus = append(menus, map[string]any{"id": section.ID, "title": section.Title, "icon": section.Icon, "action": map[string]any{"type": "url", "url": target.String()}})
		}
	}
	menus = append(menus, map[string]any{"id": "namespace-catalog", "title": "Namespace Catalog", "icon": "folder-shared", "action": map[string]any{"type": "window", "resourceUri": "window://" + config.Namespace + "/namespaces"}})
	data, err := json.Marshal(map[string]any{"schemaVersion": 1, "name": title, "icon": icon, "applications": []any{map[string]any{"id": applicationID, "title": title, "icon": icon, "menus": menus}}})
	return Definition{URI: "navigation://" + config.Namespace + "/main", JSON: data}, err
}

func embeddedNavigation(config Config) (Definition, error) {
	var app map[string]any
	if err := json.Unmarshal(config.Navigation, &app); err != nil {
		return Definition{}, err
	}
	id, _ := app["id"].(string)
	title, _ := app["title"].(string)
	icon, _ := app["icon"].(string)
	if id == "" || title == "" || icon == "" {
		return Definition{}, fmt.Errorf("invalid native application")
	}
	// Reuse the same trusted URL validation as legacy management definitions.
	check := config
	check.Navigation = nil
	check.Sections = []Section{}
	if _, err := navigation(check); err != nil {
		return Definition{}, err
	}
	base, err := url.Parse(config.PublicURL)
	if err != nil || base.Host == "" {
		return Definition{}, fmt.Errorf("native application public URL required")
	}
	var bind func([]any) error
	bind = func(items []any) error {
		for _, raw := range items {
			item, ok := raw.(map[string]any)
			if !ok {
				return fmt.Errorf("invalid native menu")
			}
			section, _ := item["section"].(string)
			if section == "" {
				section, _ = item["id"].(string)
			}
			if section == "" {
				return fmt.Errorf("invalid native menu section")
			}
			target := *base
			query := target.Query()
			query.Set("section", section)
			target.RawQuery = query.Encode()
			item["action"] = map[string]any{"type": "url", "url": target.String()}
			if children, ok := item["children"].([]any); ok {
				if err := bind(children); err != nil {
					return err
				}
			}
		}
		return nil
	}
	menus, ok := app["menus"].([]any)
	if !ok || len(menus) == 0 {
		return Definition{}, fmt.Errorf("native application menu required")
	}
	if err := bind(menus); err != nil {
		return Definition{}, err
	}
	data, err := json.Marshal(map[string]any{"schemaVersion": 1, "name": title, "icon": icon, "applications": []any{app}})
	return Definition{URI: "navigation://" + config.Namespace + "/main", JSON: data}, err
}
