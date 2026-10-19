package catalog

import (
	"encoding/json"
	"os"
	"sort"
	"strings"

	"github.com/viant/datly-studio/internal/managementcatalog"
	studioauth "github.com/viant/datly-studio/studio/auth/reader"
	namespaceext "github.com/viant/mcp-protocol/extension/namespaces"
	"github.com/viant/scy/auth/jwt"
)

// These are server-owned settings. The portable definitions contain no records
// or credentials; any verified Studio principal may discover the management UI.
// Its datasource calls still pass through the actual management tool's policy.
func management(claims *jwt.Claims, auth *studioauth.Output) (*Namespace, []managementcatalog.Definition, error) {
	if claims == nil || claims.Subject == "" || auth == nil || auth.Auth == nil || auth.Auth.Subject != claims.Subject {
		return nil, nil, fail(403, "verified Studio principal is required")
	}
	id := os.Getenv("STUDIO_MANAGEMENT_NAMESPACE")
	if id == "" {
		return nil, nil, nil
	}
	var sections []managementcatalog.Section
	if raw := os.Getenv("STUDIO_MANAGEMENT_SECTIONS_JSON"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &sections); err != nil {
			return nil, nil, fail(503, "management sections are not configured correctly")
		}
	}
	definitions, err := managementcatalog.NamespaceWindow(managementcatalog.Config{
		PublicURL:       os.Getenv("STUDIO_MANAGEMENT_PUBLIC_URL"),
		ApplicationID:   os.Getenv("STUDIO_MANAGEMENT_APPLICATION_ID"),
		ApplicationName: os.Getenv("STUDIO_MANAGEMENT_APPLICATION_NAME"),
		ApplicationIcon: os.Getenv("STUDIO_MANAGEMENT_APPLICATION_ICON"),
		Sections:        sections,
		Namespace:       id, ProviderIdentity: os.Getenv("STUDIO_MANAGEMENT_PROVIDER_ID"), MCPService: os.Getenv("STUDIO_MANAGEMENT_MCP_SERVICE"),
	})
	if err != nil {
		return nil, nil, fail(503, "management resource catalog is not configured correctly")
	}
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
		name, description := "Namespaces", "Manage Datly Studio namespaces"
		if resourceKind == "navigation" {
			name, description = "Management", "Open Studio management"
		}
		result = append(result, Reference{URI: definition.URI, Kind: resourceKind, Name: name, Description: description, MimeType: "application/json"})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].URI < result[j].URI })
	return result
}
