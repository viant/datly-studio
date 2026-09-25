package store_presence

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const UsageDatlyResourceNamespace = "studio_resource_namespaces_store_presence_usage"

//go:embed "sql/read.sql"
var UsageDatlyResources embed.FS
