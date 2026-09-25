package store_mcp_names

// NameSource is generated canonical view metadata for version.
type NameSource struct {
	ReportId     string  `sqlx:"report_id"`
	VersionNo    int     `sqlx:"version_no"`
	GeneratedDql *string `sqlx:"generated_dql"`
	AuthoredDql  *string `sqlx:"authored_dql"`
}
