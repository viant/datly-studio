package store_active_others

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const GenerationDatlyResourceNamespace = "studio_runtime_generations_store_active_others_generation"

//go:embed "sql/read.sql"
var GenerationDatlyResources embed.FS
