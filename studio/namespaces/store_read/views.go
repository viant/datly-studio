package store_read

import (
	time "time"
)

// StoredNamespace is generated canonical view metadata for namespace.
type StoredNamespace struct {
	OwnerId     string    `sqlx:"owner_id"`
	Name        string    `sqlx:"name"`
	Title       string    `sqlx:"title"`
	Description *string   `sqlx:"description"`
	Status      string    `sqlx:"status"`
	Etag        int64     `sqlx:"etag"`
	CreatedAt   time.Time `sqlx:"created_at"`
	UpdatedAt   time.Time `sqlx:"updated_at"`
}
