package store_readers

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ReaderDatlyResourceNamespace = "studio_runtime_generations_store_readers_reader"

//go:embed "sql/exposures.sql" "sql/folders.sql" "sql/read.sql" "sql/skills.sql"
var ReaderDatlyResources embed.FS
