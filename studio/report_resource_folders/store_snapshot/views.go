package store_snapshot

// SnapshotFolder is generated canonical view metadata for folder.
type SnapshotFolder struct {
	ReportId  string `sqlx:"report_id"`
	VersionNo int    `sqlx:"version_no"`
	FolderId  string `sqlx:"folder_id"`
	Namespace string `sqlx:"namespace"`
	RootPath  string `sqlx:"root_path"`
	UriPrefix string `sqlx:"uri_prefix"`
	Ordinal   int    `sqlx:"ordinal"`
}
