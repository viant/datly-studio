package store_write

import (
	time "time"
)

// StoredNamespace is generated canonical view metadata for namespace.
type StoredNamespace struct {
	OwnerId     string              `sqlx:"owner_id,primaryKey"`
	Name        string              `sqlx:"name,primaryKey"`
	Title       string              `sqlx:"title"`
	Description *string             `sqlx:"description"`
	Status      string              `sqlx:"status"`
	Etag        *int64              `writer:"concurrency" sqlx:"etag"`
	CreatedAt   *time.Time          `sqlx:"created_at"`
	UpdatedAt   *time.Time          `sqlx:"updated_at"`
	DeletedAt   *time.Time          `sqlx:"deleted_at"`
	Has         *StoredNamespaceHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredNamespaceHas"`
}

type StoredNamespaceHas struct {
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

// CurrentNamespaceView is generated canonical view metadata for namespace.
type CurrentNamespaceView struct {
	OwnerId     string     `sqlx:"owner_id,primaryKey"`
	Name        string     `sqlx:"name,primaryKey"`
	Title       string     `sqlx:"title"`
	Description *string    `sqlx:"description"`
	Status      string     `sqlx:"status"`
	Etag        *int64     `sqlx:"etag"`
	CreatedAt   *time.Time `sqlx:"created_at"`
	UpdatedAt   *time.Time `sqlx:"updated_at"`
	DeletedAt   *time.Time `sqlx:"deleted_at"`
}

type NamespaceKeysRow struct {
	OwnerId string `sqlx:"owner_id"`
	Name    string `sqlx:"name"`
}
