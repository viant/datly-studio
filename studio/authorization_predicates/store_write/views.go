package store_write

import (
	time "time"
)

// StoredAuthorizationPredicate is generated canonical view metadata for authorization_predicate.
type StoredAuthorizationPredicate struct {
	Name         string                           `sqlx:"name,primaryKey" validate:"required"`
	Title        string                           `sqlx:"title"`
	PackagePath  string                           `sqlx:"package_path"`
	TypeName     string                           `sqlx:"type_name"`
	SqlScopeJson *string                          `sqlx:"sql_scope_json"`
	OwnerId      string                           `sqlx:"owner_id"`
	Status       string                           `sqlx:"status"`
	Etag         *int                             `writer:"concurrency" sqlx:"etag"`
	Description  *string                          `sqlx:"description"`
	CreatedAt    *time.Time                       `sqlx:"created_at"`
	UpdatedAt    *time.Time                       `sqlx:"updated_at"`
	DeletedAt    *time.Time                       `sqlx:"deleted_at"`
	Has          *StoredAuthorizationPredicateHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredAuthorizationPredicateHas"`
}

type StoredAuthorizationPredicateHas struct {
	Name         bool
	Title        bool
	PackagePath  bool
	TypeName     bool
	SqlScopeJson bool
	OwnerId      bool
	Status       bool
	Etag         bool
	Description  bool
	CreatedAt    bool
	UpdatedAt    bool
	DeletedAt    bool
}

// CurrentAuthorizationPredicateView is generated canonical view metadata for authorization_predicate.
type CurrentAuthorizationPredicateView struct {
	Name         string     `sqlx:"name,primaryKey" validate:"required"`
	Title        string     `sqlx:"title"`
	PackagePath  string     `sqlx:"package_path"`
	TypeName     string     `sqlx:"type_name"`
	SqlScopeJson *string    `sqlx:"sql_scope_json"`
	OwnerId      string     `sqlx:"owner_id"`
	Status       string     `sqlx:"status"`
	Etag         *int       `sqlx:"etag"`
	Description  *string    `sqlx:"description"`
	CreatedAt    *time.Time `sqlx:"created_at"`
	UpdatedAt    *time.Time `sqlx:"updated_at"`
	DeletedAt    *time.Time `sqlx:"deleted_at"`
}

type AuthorizationPredicateKeysRow struct {
	Name string `sqlx:"name"`
}
