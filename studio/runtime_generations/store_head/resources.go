package store_head

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const HeadDatlyResourceNamespace = "studio_runtime_generations_store_head_head"

//go:embed "sql/read.sql"
var HeadDatlyResources embed.FS
