package store_global_access

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ReportDatlyResourceNamespace = "studio_reports_store_global_access_report"

//go:embed "sql/read.sql"
var ReportDatlyResources embed.FS
