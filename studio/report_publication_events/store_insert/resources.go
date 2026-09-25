package store_insert

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const EventDatlyResourceNamespace = "studio_report_publication_events_store_insert_event"

//go:embed "sql/read.sql"
var EventDatlyResources embed.FS
