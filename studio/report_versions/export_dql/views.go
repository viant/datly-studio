package export_dql

// VersionSource is generated canonical view metadata for version.
type VersionSource struct {
	ReportId  string `sqlx:"report_id"`
	VersionNo int    `sqlx:"version_no"`
	Dql       string `sqlx:"dql"`
}
