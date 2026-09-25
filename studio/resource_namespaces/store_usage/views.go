package store_usage

// NamespaceUsage is generated canonical view metadata for usage.
type NamespaceUsage struct {
	ReportId  string `sqlx:"report_id"`
	Namespace string `sqlx:"namespace"`
}
