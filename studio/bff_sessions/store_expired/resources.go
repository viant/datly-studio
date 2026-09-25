package store_expired

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const SessionDatlyResourceNamespace = "studio_bff_sessions_store_expired_session"

//go:embed "sql/read.sql"
var SessionDatlyResources embed.FS
