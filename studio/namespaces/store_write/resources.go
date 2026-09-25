package store_write

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const NamespaceDatlyResourceNamespace = "studio_namespaces_store_write_namespace"

//go:embed "sql/current_namespace.sql" "sql/namespace_keys.sql" "sql/read.sql"
var NamespaceDatlyResources embed.FS
