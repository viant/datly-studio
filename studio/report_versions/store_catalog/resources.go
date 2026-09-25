package store_catalog

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const VersionDatlyResourceNamespace = "studio_report_versions_store_catalog_version"

//go:embed "sql/read.sql"
var VersionDatlyResources embed.FS
