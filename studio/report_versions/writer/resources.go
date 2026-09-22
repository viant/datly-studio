package writer

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const VersionDatlyResourceNamespace = "studio_report_versions_writer_version"

//go:embed "sql/current_version.sql" "sql/version.sql" "sql/version_keys.sql"
var VersionDatlyResources embed.FS
