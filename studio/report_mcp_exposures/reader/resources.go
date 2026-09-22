package reader

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ExposureDatlyResourceNamespace = "studio_report_mcp_exposures_reader_exposure"

//go:embed "sql/exposure.sql"
var ExposureDatlyResources embed.FS
