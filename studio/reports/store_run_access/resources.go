package store_run_access

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const AccessDatlyResourceNamespace = "studio_reports_store_run_access_access"

//go:embed "sql/read.sql"
var AccessDatlyResources embed.FS
