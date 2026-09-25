package store_write

// StoredFolder is generated canonical view metadata for folder.
type StoredFolder struct {
	ReportId     string           `sqlx:"report_id,primaryKey"`
	VersionNo    int              `sqlx:"version_no,primaryKey"`
	FolderId     string           `sqlx:"folder_id,primaryKey"`
	Namespace    string           `sqlx:"namespace"`
	RootPath     string           `sqlx:"root_path"`
	UriPrefix    string           `sqlx:"uri_prefix"`
	Ordinal      int              `sqlx:"ordinal"`
	ShouldDelete bool             `sqlx:"-" writer:"delete"`
	Has          *StoredFolderHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredFolderHas"`
}

type StoredFolderHas struct {
	ReportId     bool
	VersionNo    bool
	FolderId     bool
	Namespace    bool
	RootPath     bool
	UriPrefix    bool
	Ordinal      bool
	ShouldDelete bool
}

// CurrentFolderView is generated canonical view metadata for folder.
type CurrentFolderView struct {
	ReportId  string `sqlx:"report_id,primaryKey"`
	VersionNo int    `sqlx:"version_no,primaryKey"`
	FolderId  string `sqlx:"folder_id,primaryKey"`
	Namespace string `sqlx:"namespace"`
	RootPath  string `sqlx:"root_path"`
	UriPrefix string `sqlx:"uri_prefix"`
	Ordinal   int    `sqlx:"ordinal"`
}

type FolderKeysRow struct {
	ReportId  string `sqlx:"report_id"`
	VersionNo int    `sqlx:"version_no"`
	FolderId  string `sqlx:"folder_id"`
}
