package writer

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const GenerationDatlyResourceNamespace = "studio_runtime_generations_writer_generation"

//go:embed "sql/current_generation.sql" "sql/generation.sql" "sql/generation_keys.sql"
var GenerationDatlyResources embed.FS
