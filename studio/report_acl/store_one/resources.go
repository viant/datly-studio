package store_one

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const AclDatlyResourceNamespace = "studio_report_acl_store_one_acl"

//go:embed "sql/read.sql"
var AclDatlyResources embed.FS
