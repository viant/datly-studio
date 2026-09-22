package writer

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ExposureDatlyResourceNamespace = "studio_report_mcp_exposures_writer_exposure"

//go:embed "sql/current_exposure.sql" "sql/exposure.sql" "sql/exposure_keys.sql"
var ExposureDatlyResources embed.FS
