package catalog

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/viant/datly-studio/internal/managementcatalog"
	"github.com/viant/datly-studio/nativeapp"
	"github.com/viant/datly-studio/runtime/managementwindow"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	identity "github.com/viant/forge/backend/resource"
	namespaceext "github.com/viant/mcp-protocol/extension/namespaces"
	"github.com/viant/scy/auth/jwt"
)

// These are server-owned settings. The portable definitions contain no records
// or credentials; any verified Studio principal may discover the management UI.
// Its datasource calls still pass through the actual management tool's policy.
type managementCatalog struct {
	namespace   *Namespace
	definitions []managementcatalog.Definition
}

// management retains the primary-catalog contract for existing integrations.
func management(claims *jwt.Claims, auth *studioauth.Output) (*Namespace, []managementcatalog.Definition, error) {
	catalogs, err := managementCatalogs(claims, auth)
	if err != nil || len(catalogs) == 0 {
		return nil, nil, err
	}
	return catalogs[0].namespace, catalogs[0].definitions, nil
}

func managementCatalogs(claims *jwt.Claims, auth *studioauth.Output) ([]managementCatalog, error) {
	if claims == nil || claims.Subject == "" || auth == nil || auth.Auth == nil || auth.Auth.Subject != claims.Subject {
		return nil, fail(403, "verified Studio principal is required")
	}
	id := os.Getenv("STUDIO_MANAGEMENT_NAMESPACE")
	managementSettings.RLock()
	installed := managementSettings.config
	registered := managementSettings.resources
	additional := append([]*managementcatalog.Config(nil), managementSettings.additional...)
	configured := installed != nil
	managementSettings.RUnlock()
	if id == "" && !configured {
		return nil, nil
	}
	var sections []managementcatalog.Section
	if raw := os.Getenv("STUDIO_MANAGEMENT_SECTIONS_JSON"); raw != "" && installed == nil {
		if err := json.Unmarshal([]byte(raw), &sections); err != nil {
			return nil, fail(503, "management sections are not configured correctly")
		}
	}
	config := managementcatalog.Config{
		PublicURL:       os.Getenv("STUDIO_MANAGEMENT_PUBLIC_URL"),
		ApplicationID:   os.Getenv("STUDIO_MANAGEMENT_APPLICATION_ID"),
		ApplicationName: os.Getenv("STUDIO_MANAGEMENT_APPLICATION_NAME"),
		ApplicationIcon: os.Getenv("STUDIO_MANAGEMENT_APPLICATION_ICON"),
		Sections:        sections,
		Namespace:       id, ProviderIdentity: os.Getenv("STUDIO_MANAGEMENT_PROVIDER_ID"), MCPService: os.Getenv("STUDIO_MANAGEMENT_MCP_SERVICE"),
	}
	if installed != nil {
		config = *installed
		id = config.Namespace
	}
	configs := append([]*managementcatalog.Config{&config}, additional...)
	catalogs := make([]managementCatalog, 0, len(configs))
	for _, current := range configs {
		namespace, definitions, err := managementForConfig(*current, registered)
		if err != nil {
			return nil, err
		}
		catalogs = append(catalogs, managementCatalog{namespace: namespace, definitions: definitions})
	}
	return catalogs, nil
}

func managementForConfig(config managementcatalog.Config, registered []managementwindow.Definition) (*Namespace, []managementcatalog.Definition, error) {
	id := config.Namespace

	definitions, err := managementcatalog.NamespaceWindow(config)
	if err != nil {
		return nil, nil, fail(503, "management resource catalog is not configured correctly")
	}
	byURI := make(map[string]managementwindow.Definition, len(definitions)+len(registered))
	for _, definition := range definitions {
		byURI[definition.URI] = definition
	}
	for _, definition := range registered {
		uri, _ := identity.ParseResourceURI(definition.URI)
		if uri.Namespace == id {
			byURI[definition.URI] = definition
		}
	}
	definitions = definitions[:0]
	for _, definition := range byURI {
		if strings.HasPrefix(definition.URI, "navigation://") {
			var app map[string]any
			if err := json.Unmarshal(definition.JSON, &app); err != nil {
				return nil, nil, err
			}
			var bind func([]any)
			bind = func(menus []any) {
				for _, raw := range menus {
					menu, ok := raw.(map[string]any)
					if !ok {
						continue
					}
					if children, ok := menu["children"].([]any); ok && len(children) > 0 {
						delete(menu, "action")
						bind(children)
						continue
					}
					section, _ := menu["section"].(string)
					if section == "" {
						section, _ = menu["id"].(string)
					}
					windowName := managementcatalog.WindowName(section)
					uri := "window://" + id + "/" + windowName
					if _, exists := byURI[uri]; exists {
						menu["action"] = map[string]any{"type": "window", "resourceUri": uri}
						delete(menu, "disabled")
					} else if action, ok := menu["action"].(map[string]any); ok && action["type"] == "url" {
						// Unported screens remain explicit pending items. They must
						// not open a separate native application from this workspace.
						delete(menu, "action")
						menu["disabled"] = true
					}
				}
			}
			applications, _ := app["applications"].([]any)
			for _, raw := range applications {
				application, ok := raw.(map[string]any)
				if !ok {
					continue
				}
				var styles []string
				for uri := range byURI {
					if strings.HasPrefix(uri, "style://"+id+"/") {
						styles = append(styles, uri)
					}
				}
				if len(styles) > 0 {
					sort.Strings(styles)
					application["styleResources"] = styles
				}
				if menus, ok := application["menus"].([]any); ok {
					bind(menus)
				}
			}
			definition.JSON, _ = json.Marshal(app)
		}
		definitions = append(definitions, managementwindow.Definition{URI: definition.URI, JSON: append(json.RawMessage(nil), definition.JSON...)})
	}
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].URI < definitions[j].URI })
	uri, err := namespaceext.URI(id)
	if err != nil {
		return nil, nil, fail(503, "management namespace identity is invalid")
	}
	return &Namespace{URI: uri, Name: id, Description: "Datly Studio management"}, definitions, nil
}

func managementReferences(definitions []managementcatalog.Definition, kind string) []Reference {
	result := []Reference{}
	for _, definition := range definitions {
		resourceKind, _, _ := strings.Cut(definition.URI, "://")
		if kind != "" && kind != resourceKind {
			continue
		}
		_, name, _ := strings.Cut(definition.URI, "://")
		_, name, _ = strings.Cut(name, "/")
		name = strings.ReplaceAll(name, "_", " ")
		if name != "" {
			name = strings.ToUpper(name[:1]) + name[1:]
		}
		description := "Datly Studio management " + resourceKind
		if resourceKind == "navigation" {
			name, description = "Management", "Open Studio management"
		}
		result = append(result, Reference{URI: definition.URI, Kind: resourceKind, Name: name, Description: description, MimeType: "application/json"})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].URI < result[j].URI })
	return result
}

var managementSettings struct {
	sync.RWMutex
	config     *managementcatalog.Config
	additional []*managementcatalog.Config
	resources  []managementwindow.Definition
}

// ConfigureManagementResources registers binary-owned management resources.
// Call before serving; authenticated catalog discovery and resource reads keep
// their existing guards. Exact URIs replace defaults for the same namespace.
// Definitions are copied and cannot be selected or supplied by a tool caller.
func ConfigureManagementResources(definitions []managementwindow.Definition) error {
	owned := make([]managementwindow.Definition, 0, len(definitions))
	seen := map[string]bool{}
	for _, definition := range definitions {
		uri, err := identity.ParseResourceURI(definition.URI)
		if err != nil || !json.Valid(definition.JSON) || seen[definition.URI] {
			return fail(400, "invalid management resource definition")
		}
		switch uri.Kind {
		case "window", "datasource", "style", "mcp":
		default:
			return fail(400, "unsupported management resource kind")
		}
		seen[definition.URI] = true
		owned = append(owned, managementwindow.Definition{URI: definition.URI, JSON: append(json.RawMessage(nil), definition.JSON...)})
	}
	managementSettings.Lock()
	managementSettings.resources = owned
	managementSettings.Unlock()
	return nil
}

// ConfigureManagementNavigation installs one binary-owned native menu. Passing
// nil selects Data Studio's embedded definition. Call before serving requests.
// PublicURL is the trusted host URL; no caller-provided navigation is accepted.
func ConfigureManagementNavigation(definition []byte, publicURL, providerIdentity string) error {
	return ConfigureManagementApplications([]ManagementApplication{{Navigation: definition, PublicURL: publicURL, ProviderIdentity: providerIdentity}})
}

// ManagementApplication is a binary-owned administrative UI. A nil Navigation
// selects the embedded Data Studio application; definitions never come from callers.
type ManagementApplication struct {
	Navigation       []byte
	PublicURL        string
	ProviderIdentity string
}

// ConfigureManagementApplications installs isolated administrative namespaces
// atomically before serving requests. Each application owns its menu, built-in
// resources, producer connection and namespace-scoped resource overrides.
// Existing principal and operation authorization applies to every application.
func ConfigureManagementApplications(applications []ManagementApplication) error {
	if len(applications) == 0 {
		return fail(400, "management applications required")
	}
	configs := make([]*managementcatalog.Config, 0, len(applications))
	seen := map[string]bool{}
	for _, application := range applications {
		definition := application.Navigation
		if len(definition) == 0 {
			definition = nativeapp.Navigation()
		}
		var app struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(definition, &app); err != nil {
			return err
		}
		if seen[app.ID] {
			return fail(400, "duplicate management namespace")
		}
		config := managementcatalog.Config{Namespace: app.ID, ApplicationID: app.ID, ProviderIdentity: application.ProviderIdentity, MCPService: "mcp://" + app.ID + "/admin", PublicURL: application.PublicURL, Navigation: append(json.RawMessage(nil), definition...)}
		if _, err := managementcatalog.NamespaceWindow(config); err != nil {
			return err
		}
		seen[app.ID] = true
		configs = append(configs, &config)
	}
	managementSettings.Lock()
	managementSettings.config = configs[0]
	managementSettings.additional = configs[1:]
	managementSettings.Unlock()
	return nil
}
