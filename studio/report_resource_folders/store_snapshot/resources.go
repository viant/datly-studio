package store_snapshot

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const FolderDatlyResourceNamespace = "studio_report_resource_folders_store_snapshot_folder"

//go:embed "sql/read.sql"
var FolderDatlyResources embed.FS
