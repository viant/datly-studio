package store_write

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const WarmupRunDatlyResourceNamespace = "studio_report_warmup_runs_store_write_warmup_run"

//go:embed "sql/current_warmup_run.sql" "sql/read.sql" "sql/warmup_run_keys.sql"
var WarmupRunDatlyResources embed.FS
