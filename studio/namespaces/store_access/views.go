package store_access

// StoredNamespace is generated canonical view metadata for namespace.
type StoredNamespace struct {
	NamespaceId      string  `sqlx:"namespace_id" json:"namespaceId"`
	Visibility       string  `sqlx:"visibility" json:"visibility"`
	AllowedRolesJson *string `sqlx:"allowed_roles_json" json:"-"`
	McpEnabled       bool    `sqlx:"mcp_enabled" json:"mcpEnabled"`
	McpPort          *int    `sqlx:"mcp_port" json:"mcpPort,omitempty"`
	OwnerId          string  `sqlx:"owner_id"`
	Name             string  `sqlx:"name"`
}
