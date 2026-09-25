package list

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const EventDatlyResourceNamespace = "studio_report_publication_events_list_event"

//go:embed "sql/event.sql"
var EventDatlyResources embed.FS
