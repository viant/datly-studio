package reader

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const AclDatlyResourceNamespace = "studio_report_acl_reader_acl"

//go:embed "sql/acl.sql"
var AclDatlyResources embed.FS
