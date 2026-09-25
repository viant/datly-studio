package store_active

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const DefinitionDatlyResourceNamespace = "studio_runtime_generations_store_active_definition"

//go:embed "sql/read.sql"
var DefinitionDatlyResources embed.FS
