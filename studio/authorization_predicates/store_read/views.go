package store_read

import (
	time "time"
)

// StoredAuthorizationPredicate is generated canonical view metadata for authorization_predicate.
type StoredAuthorizationPredicate struct {
	Name         string     `sqlx:"name"`
	Title        string     `sqlx:"title"`
	PackagePath  string     `sqlx:"package_path"`
	TypeName     string     `sqlx:"type_name"`
	OwnerId      string     `sqlx:"owner_id"`
	Status       string     `sqlx:"status"`
	Etag         int64      `sqlx:"etag"`
	Description  *string    `sqlx:"description"`
	SqlScopeJson *string    `sqlx:"sql_scope_json"`
	CreatedAt    *time.Time `sqlx:"created_at"`
	UpdatedAt    *time.Time `sqlx:"updated_at"`
}
