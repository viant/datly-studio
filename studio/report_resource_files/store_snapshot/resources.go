package store_snapshot

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const FileDatlyResourceNamespace = "studio_report_resource_files_store_snapshot_file"

//go:embed "sql/read.sql"
var FileDatlyResources embed.FS
