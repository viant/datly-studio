package reader

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ParameterDatlyResourceNamespace = "studio_report_parameters_reader_parameter"

//go:embed "sql/parameter.sql" "sql/predicates.sql"
var ParameterDatlyResources embed.FS
