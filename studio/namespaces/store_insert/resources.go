package store_insert

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const NamespaceDatlyResourceNamespace = "studio_namespaces_store_insert_namespace"

//go:embed "sql/read.sql"
var NamespaceDatlyResources embed.FS
