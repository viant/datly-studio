package store_status

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ConnectorDatlyResourceNamespace = "studio_connectors_store_status_connector"

//go:embed "sql/connector_keys.sql" "sql/current_connector.sql" "sql/read.sql"
var ConnectorDatlyResources embed.FS
