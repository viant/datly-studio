package create

import (
	time "time"
)

// NamespaceRecord is generated canonical view metadata for namespace.
type NamespaceRecord struct {
	OwnerId     string              `sqlx:"owner_id,primaryKey" json:"ownerId,omitempty"`
	Name        string              `sqlx:"name,primaryKey" json:"name"`
	Title       string              `json:"title" sqlx:"title"`
	Description *string             `json:"description,omitempty" sqlx:"description"`
	Status      string              `json:"-" sqlx:"status"`
	Etag        *int64              `json:"-" sqlx:"etag"`
	CreatedAt   *time.Time          `json:"-" sqlx:"created_at"`
	UpdatedAt   *time.Time          `json:"-" sqlx:"updated_at"`
	DeletedAt   *time.Time          `json:"-" sqlx:"deleted_at"`
	Has         *NamespaceRecordHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"NamespaceRecordHas"`
}

type NamespaceRecordHas struct {
	OwnerId     bool
	Name        bool
	Title       bool
	Description bool
	Status      bool
	Etag        bool
	CreatedAt   bool
	UpdatedAt   bool
	DeletedAt   bool
}
