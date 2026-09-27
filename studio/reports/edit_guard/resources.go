package edit_guard

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ReportDatlyResourceNamespace = "studio_reports_edit_guard_report"

//go:embed "sql/report.sql"
var ReportDatlyResources embed.FS
