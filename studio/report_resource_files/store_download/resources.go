package store_download

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const FileDatlyResourceNamespace = "studio_report_resource_files_store_download_file"

//go:embed "sql/read.sql"
var FileDatlyResources embed.FS
