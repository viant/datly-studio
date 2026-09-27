package descriptor

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const VersionDatlyResourceNamespace = "studio_report_versions_descriptor_version"

//go:embed "sql/version.sql"
var VersionDatlyResources embed.FS
