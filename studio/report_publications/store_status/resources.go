package store_status

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const PublicationDatlyResourceNamespace = "studio_report_publications_store_status_publication"

//go:embed "sql/read.sql"
var PublicationDatlyResources embed.FS
