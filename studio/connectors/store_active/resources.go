package store_active

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ConnectorDatlyResourceNamespace = "studio_connectors_store_active_connector"

//go:embed "sql/read.sql"
var ConnectorDatlyResources embed.FS
