package reader

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const FileDatlyResourceNamespace = "studio_report_resource_files_reader_file"

//go:embed "sql/file.sql"
var FileDatlyResources embed.FS
