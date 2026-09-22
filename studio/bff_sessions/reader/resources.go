package reader

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const SessionDatlyResourceNamespace = "studio_bff_sessions_reader_session"

//go:embed "sql/read.sql"
var SessionDatlyResources embed.FS
