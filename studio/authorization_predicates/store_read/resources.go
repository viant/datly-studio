package store_read

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const AuthorizationPredicateDatlyResourceNamespace = "studio_authorization_predicates_store_read_authorization_predicate"

//go:embed "sql/read.sql"
var AuthorizationPredicateDatlyResources embed.FS
