package store_draft_pointer

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ReportDatlyResourceNamespace = "studio_reports_store_draft_pointer_report"

//go:embed "sql/current_report.sql" "sql/read.sql" "sql/report_keys.sql"
var ReportDatlyResources embed.FS
