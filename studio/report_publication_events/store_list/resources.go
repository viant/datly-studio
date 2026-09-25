package store_list

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const EventDatlyResourceNamespace = "studio_report_publication_events_store_list_event"

//go:embed "sql/read.sql"
var EventDatlyResources embed.FS
