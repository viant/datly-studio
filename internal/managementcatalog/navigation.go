package managementcatalog

import (
	"encoding/json"
	"fmt"
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

var defaultSections = []Section{
	{"overview", "Overview", "home"}, {"connectors", "Connectors", "database"},
	{"schema", "Schema Browser", "diagram-tree"}, {"namespaces", "Namespaces", "folder-shared"},
	{"security", "Security", "shield"}, {"runtime", "Runtime", "pulse"},
	{"skills", "Skills", "learning"}, {"components", "Components", "application"},
}

func navigation(config Config) (Definition, error) {
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
			sections = defaultSections
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
