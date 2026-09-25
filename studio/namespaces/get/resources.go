package get

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const NamespaceDatlyResourceNamespace = "studio_namespaces_get_namespace"

//go:embed "sql/namespace.sql"
var NamespaceDatlyResources embed.FS
