package store_read

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const GrantDatlyResourceNamespace = "studio_authorization_grants_store_read_grant"

//go:embed "sql/read.sql"
var GrantDatlyResources embed.FS
