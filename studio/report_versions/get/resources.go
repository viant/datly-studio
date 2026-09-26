package get

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const VersionDatlyResourceNamespace = "studio_report_versions_get_version"

//go:embed "sql/version.sql"
var VersionDatlyResources embed.FS
