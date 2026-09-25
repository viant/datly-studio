package store_capabilities

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const CapabilityDatlyResourceNamespace = "studio_reports_store_capabilities_capability"

//go:embed "sql/read.sql"
var CapabilityDatlyResources embed.FS
