package writer

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const AclDatlyResourceNamespace = "studio_report_acl_writer_acl"

//go:embed "sql/acl.sql" "sql/acl_keys.sql" "sql/current_acl.sql"
var AclDatlyResources embed.FS
