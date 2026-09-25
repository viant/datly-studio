package store_mcp_names

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const VersionDatlyResourceNamespace = "studio_report_versions_store_mcp_names_version"

//go:embed "sql/read.sql"
var VersionDatlyResources embed.FS
