package reader

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const FolderDatlyResourceNamespace = "studio_report_resource_folders_reader_folder"

//go:embed "sql/folder.sql"
var FolderDatlyResources embed.FS
