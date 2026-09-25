package store_edit

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const VersionDatlyResourceNamespace = "studio_report_versions_store_edit_version"

//go:embed "sql/current_version.sql" "sql/read.sql" "sql/version_keys.sql"
var VersionDatlyResources embed.FS
