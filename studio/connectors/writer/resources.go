package writer

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ConnectorDatlyResourceNamespace = "studio_connectors_writer_connector"

//go:embed "sql/connector.sql" "sql/connector_keys.sql" "sql/current_connector.sql"
var ConnectorDatlyResources embed.FS
