package writer

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ReportDatlyResourceNamespace = "studio_reports_writer_report"

//go:embed "sql/current_report.sql" "sql/report.sql" "sql/report_keys.sql"
var ReportDatlyResources embed.FS
