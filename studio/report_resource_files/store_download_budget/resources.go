package store_download_budget

import "embed"

// DatlyResourceNamespace identifies this package's generated resource filesystem.
const BudgetDatlyResourceNamespace = "studio_report_resource_files_store_download_budget_budget"

//go:embed "sql/budget.sql"
var BudgetDatlyResources embed.FS
