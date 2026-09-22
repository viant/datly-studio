package writer

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const SessionDatlyResourceNamespace = "studio_bff_sessions_writer_session"

//go:embed "sql/current_session.sql" "sql/patch.sql" "sql/session_keys.sql"
var SessionDatlyResources embed.FS
