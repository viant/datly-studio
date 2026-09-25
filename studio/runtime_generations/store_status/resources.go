package store_status

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const GenerationDatlyResourceNamespace = "studio_runtime_generations_store_status_generation"

//go:embed "sql/read.sql"
var GenerationDatlyResources embed.FS
