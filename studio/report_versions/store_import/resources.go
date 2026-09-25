package store_import

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const VersionDatlyResourceNamespace = "studio_report_versions_store_import_version"

//go:embed "sql/file.sql" "sql/read.sql"
var VersionDatlyResources embed.FS
