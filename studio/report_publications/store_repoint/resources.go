package store_repoint

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const PublicationDatlyResourceNamespace = "studio_report_publications_store_repoint_publication"

//go:embed "sql/current_publication.sql" "sql/publication_keys.sql" "sql/read.sql"
var PublicationDatlyResources embed.FS
