package reader

// AuthorizationPredicateRecord is generated canonical view metadata for authorization_predicate.
type AuthorizationPredicateRecord struct {
	Name         string `sqlx:"name"`
	Title        string `sqlx:"title"`
	PackagePath  string `sqlx:"package_path"`
	TypeName     string `sqlx:"type_name"`
	SqlScopeJson any    `sqlx:"sql_scope_json,enc=JSON"`
	OwnerId      string `sqlx:"owner_id"`
	Status       string `sqlx:"status"`
}
