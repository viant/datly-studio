package store_catalog

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const GenerationDatlyResourceNamespace = "studio_runtime_generations_store_catalog_generation"

//go:embed "sql/read.sql"
var GenerationDatlyResources embed.FS
