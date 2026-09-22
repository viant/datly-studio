package reader

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ReportDatlyResourceNamespace = "studio_reports_reader_report"

//go:embed "sql/report.sql"
var ReportDatlyResources embed.FS
