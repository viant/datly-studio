package store_write

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ClaimDatlyResourceNamespace = "studio_resource_namespace_claims_store_write_claim"

//go:embed "sql/claim_keys.sql" "sql/current_claim.sql" "sql/read.sql"
var ClaimDatlyResources embed.FS
