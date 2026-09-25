package store_preview_scoped

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ConnectorDatlyResourceNamespace = "studio_connectors_store_preview_scoped_connector"

//go:embed "sql/read.sql"
var ConnectorDatlyResources embed.FS
