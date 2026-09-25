package store_state

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const GenerationDatlyResourceNamespace = "studio_runtime_generations_store_state_generation"

//go:embed "sql/current_generation.sql" "sql/generation_keys.sql" "sql/read.sql"
var GenerationDatlyResources embed.FS
