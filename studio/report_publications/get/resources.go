package get

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const PublicationDatlyResourceNamespace = "studio_report_publications_get_publication"

//go:embed "sql/publication.sql"
var PublicationDatlyResources embed.FS
