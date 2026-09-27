package store_download_budget

// Budget is generated canonical view metadata for budget.
type Budget struct {
	FileCount    int64 `sqlx:"file_count"`
	ContentBytes int64 `sqlx:"content_bytes"`
}
