package reader

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ConnectorDatlyResourceNamespace = "studio_connectors_reader_connector"

//go:embed "sql/connector.sql"
var ConnectorDatlyResources embed.FS
