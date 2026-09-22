package reader

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const GenerationDatlyResourceNamespace = "studio_runtime_generations_reader_generation"

//go:embed "sql/generation.sql"
var GenerationDatlyResources embed.FS
