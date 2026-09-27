package store_insert

import (
	time "time"
)

// StoredAuthorizationPredicate is generated canonical view metadata for authorization_predicate.
type StoredAuthorizationPredicate struct {
	Name         string                           `sqlx:"name,primaryKey"`
	Title        string                           `sqlx:"title"`
	Description  *string                          `sqlx:"description"`
	PackagePath  string                           `sqlx:"package_path"`
	TypeName     string                           `sqlx:"type_name"`
	SqlScopeJson *string                          `sqlx:"sql_scope_json"`
	OwnerId      string                           `sqlx:"owner_id"`
	Status       string                           `sqlx:"status"`
	Etag         int64                            `sqlx:"etag"`
	CreatedAt    time.Time                        `sqlx:"created_at"`
	UpdatedAt    time.Time                        `sqlx:"updated_at"`
	Has          *StoredAuthorizationPredicateHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredAuthorizationPredicateHas"`
}

type StoredAuthorizationPredicateHas struct {
	Name         bool
	Title        bool
	Description  bool
	PackagePath  bool
	TypeName     bool
	SqlScopeJson bool
	OwnerId      bool
	Status       bool
	Etag         bool
	CreatedAt    bool
	UpdatedAt    bool
}
