package writer

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ConfigDatlyResourceNamespace = "studio_report_cube_configs_writer_config"

//go:embed "sql/config.sql" "sql/config_keys.sql" "sql/current_config.sql"
var ConfigDatlyResources embed.FS
