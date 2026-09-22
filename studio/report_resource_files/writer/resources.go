package writer

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const FileDatlyResourceNamespace = "studio_report_resource_files_writer_file"

//go:embed "sql/current_file.sql" "sql/file.sql" "sql/file_keys.sql"
var FileDatlyResources embed.FS
