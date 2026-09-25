package store_write

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const FolderDatlyResourceNamespace = "studio_report_resource_folders_store_write_folder"

//go:embed "sql/current_folder.sql" "sql/folder_keys.sql" "sql/read.sql"
var FolderDatlyResources embed.FS
