package reader

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ViewDatlyResourceNamespace = "studio_report_views_reader_view"

//go:embed "sql/fields.sql" "sql/view.sql"
var ViewDatlyResources embed.FS
