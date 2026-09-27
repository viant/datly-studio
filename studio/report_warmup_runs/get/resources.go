package get

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const WarmupRunDatlyResourceNamespace = "studio_report_warmup_runs_get_warmup_run"

//go:embed "sql/warmup_run.sql"
var WarmupRunDatlyResources embed.FS
