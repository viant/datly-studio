package writer

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const FolderDatlyResourceNamespace = "studio_report_resource_folders_writer_folder"

//go:embed "sql/current_folder.sql" "sql/folder.sql" "sql/folder_keys.sql"
var FolderDatlyResources embed.FS
