package store_candidate

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const DefinitionDatlyResourceNamespace = "studio_runtime_generations_store_candidate_definition"

//go:embed "sql/read.sql"
var DefinitionDatlyResources embed.FS
