package store_owner

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const ReportDatlyResourceNamespace = "studio_report_publication_events_store_owner_report"

//go:embed "sql/read.sql"
var ReportDatlyResources embed.FS
