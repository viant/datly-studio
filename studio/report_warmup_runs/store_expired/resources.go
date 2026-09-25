package store_expired

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const WarmupRunDatlyResourceNamespace = "studio_report_warmup_runs_store_expired_warmup_run"

//go:embed "sql/read.sql"
var WarmupRunDatlyResources embed.FS
