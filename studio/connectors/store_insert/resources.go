package store_insert

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ConnectorDatlyResourceNamespace = "studio_connectors_store_insert_connector"

//go:embed "sql/read.sql"
var ConnectorDatlyResources embed.FS
