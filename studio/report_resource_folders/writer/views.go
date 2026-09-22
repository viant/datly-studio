package writer

// ReportResourceFolder is generated canonical view metadata for folder.
type ReportResourceFolder struct {
	ReportId *string `sqlx:"report_id,primaryKey,refTable=report_versions,refColumn=report_id,required=true" validate:"required"`
	VersionNo *int `sqlx:"version_no,primaryKey,refTable=report_versions,refColumn=version_no,required=true"`
	FolderId *string `sqlx:"folder_id,primaryKey,required=true" validate:"required"`
	Namespace *string `validate:"required" sqlx:"namespace,required=true"`
	RootPath *string `validate:"required" sqlx:"root_path,required=true"`
	UriPrefix *string `validate:"required" sqlx:"uri_prefix,required=true"`
	ShouldDelete bool `sqlx:"-" writer:"delete"`
	Ordinal *int `sqlx:"ordinal,required=true"`
	Has *ReportResourceFolderHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"ReportResourceFolderHas"`
}

type ReportResourceFolderHas struct {
	ReportId bool
	VersionNo bool
	FolderId bool
	Namespace bool
	RootPath bool
	UriPrefix bool
	ShouldDelete bool
	Ordinal bool
}

// CurrentFolderView is generated canonical view metadata for folder.
type CurrentFolderView struct {
	ReportId *string `sqlx:"report_id,primaryKey,refTable=report_versions,refColumn=report_id,required=true" validate:"required"`
	VersionNo *int `sqlx:"version_no,primaryKey,refTable=report_versions,refColumn=version_no,required=true"`
	FolderId *string `sqlx:"folder_id,primaryKey,required=true" validate:"required"`
	Namespace *string `validate:"required" sqlx:"namespace,required=true"`
	RootPath *string `validate:"required" sqlx:"root_path,required=true"`
	UriPrefix *string `validate:"required" sqlx:"uri_prefix,required=true"`
	Ordinal *int `sqlx:"ordinal,required=true"`
}

type FolderKeysRow struct {
	ReportId *string `sqlx:"report_id"`
	VersionNo *int `sqlx:"version_no"`
	FolderId *string `sqlx:"folder_id"`
}
