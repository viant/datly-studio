package reader

import "embed"

// PublicationEventDatlyResourceNamespace identifies this reader's generated
// Datly resource filesystem.
const PublicationEventDatlyResourceNamespace = "studio_report_publication_events_reader_event"

//go:embed "sql/read.sql"
var PublicationEventDatlyResources embed.FS
