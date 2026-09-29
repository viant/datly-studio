package create

import (
	time "time"
)

// NamespaceRecord is generated canonical view metadata for namespace.
type NamespaceRecord struct {
	AllowedRoles     []string            `json:"allowedRoles,omitempty" sqlx:"-"`
	NamespaceId      string              `sqlx:"namespace_id" json:"-"`
	Visibility       string              `sqlx:"visibility" json:"visibility"`
	AllowedRolesJson *string             `sqlx:"allowed_roles_json" json:"-"`
	McpEnabled       bool                `sqlx:"mcp_enabled" json:"mcpEnabled"`
	McpPort          *int                `sqlx:"mcp_port" json:"mcpPort,omitempty"`
	OwnerId          string              `sqlx:"owner_id,primaryKey" json:"ownerId,omitempty"`
	Name             string              `sqlx:"name,primaryKey" json:"name"`
	Title            string              `json:"title" sqlx:"title"`
	Description      *string             `json:"description,omitempty" sqlx:"description"`
	Status           string              `json:"-" sqlx:"status"`
	Etag             *int64              `json:"-" sqlx:"etag"`
	CreatedAt        *time.Time          `json:"-" sqlx:"created_at"`
	UpdatedAt        *time.Time          `json:"-" sqlx:"updated_at"`
	DeletedAt        *time.Time          `json:"-" sqlx:"deleted_at"`
	Has              *NamespaceRecordHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"NamespaceRecordHas"`
}

type NamespaceRecordHas struct {
	NamespaceId      bool
	Visibility       bool
	AllowedRolesJson bool
	McpEnabled       bool
	McpPort          bool
	OwnerId          bool
	Name             bool
	Title            bool
	Description      bool
	Status           bool
	Etag             bool
	CreatedAt        bool
	UpdatedAt        bool
	DeletedAt        bool
}
