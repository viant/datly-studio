package store_write

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const AclDatlyResourceNamespace = "studio_report_acl_store_write_acl"

//go:embed "sql/acl_keys.sql" "sql/current_acl.sql" "sql/patch.sql"
var AclDatlyResources embed.FS
