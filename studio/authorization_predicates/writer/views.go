package writer

import (
	time "time"
)

// AuthorizationPredicateMutationRecord is generated canonical view metadata for authorization_predicate.
type AuthorizationPredicateMutationRecord struct {
	Name         string                                   `sqlx:"name,primaryKey" validate:"required,regexp(^[a-z][a-z0-9_]*(\\.[a-z][a-z0-9_]*)+$)"`
	Title        string                                   `validate:"required" sqlx:"title"`
	PackagePath  string                                   `validate:"required" sqlx:"package_path"`
	TypeName     string                                   `validate:"required" sqlx:"type_name"`
	SqlScopeJson *string                                  `sqlx:"sql_scope_json,enc=JSON"`
	OwnerId      string                                   `sqlx:"owner_id"`
	Status       string                                   `validate:"required,choice(active,disabled)" sqlx:"status"`
	ShouldDelete bool                                     `sqlx:"-" writer:"delete"`
	Etag         *int                                     `writer:"concurrency" sqlx:"etag"`
	Description  *string                                  `sqlx:"description"`
	CreatedAt    *time.Time                               `sqlx:"created_at"`
	UpdatedAt    *time.Time                               `sqlx:"updated_at"`
	DeletedAt    *time.Time                               `sqlx:"deleted_at"`
	Has          *AuthorizationPredicateMutationRecordHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"AuthorizationPredicateMutationRecordHas"`
}

type AuthorizationPredicateMutationRecordHas struct {
	Name         bool
	Title        bool
	PackagePath  bool
	TypeName     bool
	SqlScopeJson bool
	OwnerId      bool
	Status       bool
	ShouldDelete bool
	Etag         bool
	Description  bool
	CreatedAt    bool
	UpdatedAt    bool
	DeletedAt    bool
}

// CurrentAuthorizationPredicateView is generated canonical view metadata for authorization_predicate.
type CurrentAuthorizationPredicateView struct {
	Name         string     `sqlx:"name,primaryKey" validate:"required,regexp(^[a-z][a-z0-9_]*(\\.[a-z][a-z0-9_]*)+$)"`
	Title        string     `validate:"required" sqlx:"title"`
	PackagePath  string     `validate:"required" sqlx:"package_path"`
	TypeName     string     `validate:"required" sqlx:"type_name"`
	SqlScopeJson *string    `sqlx:"sql_scope_json,enc=JSON"`
	OwnerId      string     `sqlx:"owner_id"`
	Status       string     `validate:"required,choice(active,disabled)" sqlx:"status"`
	Etag         *int       `sqlx:"etag"`
	Description  *string    `sqlx:"description"`
	CreatedAt    *time.Time `sqlx:"created_at"`
	UpdatedAt    *time.Time `sqlx:"updated_at"`
	DeletedAt    *time.Time `sqlx:"deleted_at"`
}

type AuthorizationPredicateKeysRow struct {
	Name string `sqlx:"name"`
}
