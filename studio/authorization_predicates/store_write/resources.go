package store_write

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const AuthorizationPredicateDatlyResourceNamespace = "studio_authorization_predicates_store_write_authorization_predicate"

//go:embed "sql/authorization_predicate_keys.sql" "sql/current_authorization_predicate.sql" "sql/patch.sql"
var AuthorizationPredicateDatlyResources embed.FS
