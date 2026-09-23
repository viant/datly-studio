package writer

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const PolicyDatlyResourceNamespace = "studio_resource_policy_writer_policy"

//go:embed "sql/current_history.sql" "sql/current_policy.sql" "sql/history.sql" "sql/policy.sql" "sql/policy_keys.sql"
var PolicyDatlyResources embed.FS
