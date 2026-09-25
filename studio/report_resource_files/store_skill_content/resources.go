package store_skill_content

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const FileDatlyResourceNamespace = "studio_report_resource_files_store_skill_content_file"

//go:embed "sql/read.sql"
var FileDatlyResources embed.FS
