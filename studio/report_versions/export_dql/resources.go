package export_dql

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const VersionDatlyResourceNamespace = "studio_report_versions_export_dql_version"

//go:embed "sql/version.sql"
var VersionDatlyResources embed.FS
