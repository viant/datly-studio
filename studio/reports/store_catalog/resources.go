package store_catalog

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ReportDatlyResourceNamespace = "studio_reports_store_catalog_report"

//go:embed "sql/read.sql"
var ReportDatlyResources embed.FS
