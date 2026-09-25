package store_download

// DownloadResourceFile is generated canonical view metadata for file.
type DownloadResourceFile struct {
	ReportId     string `sqlx:"report_id"`
	VersionNo    int    `sqlx:"version_no"`
	ResourcePath string `sqlx:"resource_path"`
	Content      []byte `sqlx:"content"`
}
