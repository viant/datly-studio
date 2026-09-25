package store_insert

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const PublicationDatlyResourceNamespace = "studio_report_publications_store_insert_publication"

//go:embed "sql/read.sql"
var PublicationDatlyResources embed.FS
