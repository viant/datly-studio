package writer

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ParameterDatlyResourceNamespace = "studio_report_parameters_writer_parameter"

//go:embed "sql/current_parameter.sql" "sql/current_predicate.sql" "sql/parameter.sql" "sql/parameter_keys.sql" "sql/predicate.sql"
var ParameterDatlyResources embed.FS
