package writer

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const NamespaceDatlyResourceNamespace = "studio_namespaces_writer_namespace"

//go:embed "sql/current_namespace.sql" "sql/namespace.sql" "sql/namespace_keys.sql"
var NamespaceDatlyResources embed.FS
