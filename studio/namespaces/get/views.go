package get

import (
	time "time"
)

// NamespaceRecord is generated canonical view metadata for namespace.
type NamespaceRecord struct {
	AllowedRoles     []string  `json:"allowedRoles" sqlx:"-"`
	NamespaceId      string    `sqlx:"namespace_id" json:"namespaceId"`
	Visibility       string    `sqlx:"visibility" json:"visibility"`
	AllowedRolesJson *string   `sqlx:"allowed_roles_json" json:"-"`
	McpEnabled       bool      `sqlx:"mcp_enabled" json:"mcpEnabled"`
	McpPort          *int      `sqlx:"mcp_port" json:"mcpPort,omitempty"`
	OwnerId          string    `sqlx:"owner_id"`
	Name             string    `sqlx:"name"`
	Title            string    `sqlx:"title"`
	Description      *string   `sqlx:"description"`
	Status           string    `sqlx:"status"`
	Etag             int64     `sqlx:"etag"`
	CreatedAt        time.Time `sqlx:"created_at"`
	UpdatedAt        time.Time `sqlx:"updated_at"`
}
