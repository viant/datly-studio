package reader

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ContextDatlyResourceNamespace = "studio_auth_reader_context"

//go:embed "sql/context.sql"
var ContextDatlyResources embed.FS
