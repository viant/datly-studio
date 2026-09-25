package store_snapshot

// SnapshotFile is generated canonical view metadata for file.
type SnapshotFile struct {
	ReportId      string  `sqlx:"report_id"`
	VersionNo     int     `sqlx:"version_no"`
	ResourceId    string  `sqlx:"resource_id"`
	Namespace     string  `sqlx:"namespace"`
	ResourcePath  string  `sqlx:"resource_path"`
	MediaType     *string `sqlx:"media_type"`
	Content       []byte  `sqlx:"content"`
	ContentSize   int64   `sqlx:"content_size"`
	ContentSha256 string  `sqlx:"content_sha256"`
	IsBinary      bool    `sqlx:"is_binary"`
}
