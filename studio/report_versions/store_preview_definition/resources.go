package store_preview_definition

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const DefinitionDatlyResourceNamespace = "studio_report_versions_store_preview_definition_definition"

//go:embed "sql/read.sql"
var DefinitionDatlyResources embed.FS
