package reader

import (
	time "time"
)

// NamespaceRecord is generated canonical view metadata for namespace.
type NamespaceRecord struct {
	OwnerId     string     `sqlx:"owner_id"`
	Name        string     `sqlx:"name"`
	Title       string     `sqlx:"title"`
	Status      string     `sqlx:"status"`
	Description *string    `sqlx:"description"`
	Etag        *int       `sqlx:"etag"`
	CreatedAt   *time.Time `sqlx:"created_at"`
	UpdatedAt   *time.Time `sqlx:"updated_at"`
}
