package store_write

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const FileDatlyResourceNamespace = "studio_report_resource_files_store_write_file"

//go:embed "sql/current_file.sql" "sql/file_keys.sql" "sql/read.sql"
var FileDatlyResources embed.FS
