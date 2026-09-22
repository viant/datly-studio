package writer

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const PublicationDatlyResourceNamespace = "studio_report_publications_writer_publication"

//go:embed "sql/current_publication.sql" "sql/patch.sql" "sql/publication_keys.sql"
var PublicationDatlyResources embed.FS
