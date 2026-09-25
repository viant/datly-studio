package store_insert

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const GenerationDatlyResourceNamespace = "studio_runtime_generations_store_insert_generation"

//go:embed "sql/read.sql"
var GenerationDatlyResources embed.FS
