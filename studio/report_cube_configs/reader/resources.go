package reader

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ConfigDatlyResourceNamespace = "studio_report_cube_configs_reader_config"

//go:embed "sql/config.sql"
var ConfigDatlyResources embed.FS
