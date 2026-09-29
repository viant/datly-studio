package presence

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const HeadDatlyResourceNamespace = "studio_resource_policy_presence_head"

//go:embed "sql/read.sql"
var HeadDatlyResources embed.FS
