package writer

import (
	time "time"
)

// NamespaceMutationRecord is generated canonical view metadata for namespace.
type NamespaceMutationRecord struct {
	OwnerId      string                      `sqlx:"owner_id,primaryKey" validate:"required"`
	Name         string                      `sqlx:"name,primaryKey" validate:"required,regexp(^[a-z][a-z0-9_]*(\\.[a-z][a-z0-9_]*)*$)"`
	Title        string                      `validate:"required" sqlx:"title"`
	Status       string                      `validate:"required,choice(active,archived)" sqlx:"status"`
	ShouldDelete bool                        `sqlx:"-" writer:"delete"`
	Etag         *int                        `writer:"concurrency" sqlx:"etag"`
	Description  *string                     `sqlx:"description"`
	CreatedAt    *time.Time                  `sqlx:"created_at"`
	UpdatedAt    *time.Time                  `sqlx:"updated_at"`
	DeletedAt    *time.Time                  `sqlx:"deleted_at"`
	Has          *NamespaceMutationRecordHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"NamespaceMutationRecordHas"`
}

type NamespaceMutationRecordHas struct {
	OwnerId      bool
	Name         bool
	Title        bool
	Status       bool
	ShouldDelete bool
	Etag         bool
	Description  bool
	CreatedAt    bool
	UpdatedAt    bool
	DeletedAt    bool
}

// CurrentNamespaceView is generated canonical view metadata for namespace.
type CurrentNamespaceView struct {
	OwnerId     string     `sqlx:"owner_id,primaryKey" validate:"required"`
	Name        string     `sqlx:"name,primaryKey" validate:"required,regexp(^[a-z][a-z0-9_]*(\\.[a-z][a-z0-9_]*)*$)"`
	Title       string     `validate:"required" sqlx:"title"`
	Status      string     `validate:"required,choice(active,archived)" sqlx:"status"`
	Etag        *int       `sqlx:"etag"`
	Description *string    `sqlx:"description"`
	CreatedAt   *time.Time `sqlx:"created_at"`
	UpdatedAt   *time.Time `sqlx:"updated_at"`
	DeletedAt   *time.Time `sqlx:"deleted_at"`
}

type NamespaceKeysRow struct {
	OwnerId string `sqlx:"owner_id"`
	Name    string `sqlx:"name"`
}
