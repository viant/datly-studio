package writer

import (
	time "time"
)

// ReportResourceFile is generated canonical view metadata for file.
type ReportResourceFile struct {
	ReportId *string `sqlx:"report_id,primaryKey,refTable=report_versions,refColumn=report_id,required=true" validate:"required"`
	VersionNo *int `sqlx:"version_no,primaryKey,refTable=report_versions,refColumn=version_no,required=true"`
	ResourceId *string `sqlx:"resource_id,primaryKey,required=true" validate:"required"`
	Namespace *string `validate:"required" sqlx:"namespace,required=true"`
	ResourcePath *string `validate:"required" sqlx:"resource_path,required=true"`
	Content *string `validate:"required" sqlx:"content,required=true"`
	ContentSize *int `validate:"gte=0" sqlx:"content_size,required=true"`
	ContentSha256 *string `validate:"required" sqlx:"content_sha256,required=true"`
	CreatedAt *time.Time `validate:"required" sqlx:"created_at,required=true"`
	ShouldDelete bool `sqlx:"-" writer:"delete"`
	MediaType *string `sqlx:"media_type"`
	IsBinary *int `sqlx:"is_binary,required=true"`
	Has *ReportResourceFileHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"ReportResourceFileHas"`
}

type ReportResourceFileHas struct {
	ReportId bool
	VersionNo bool
	ResourceId bool
	Namespace bool
	ResourcePath bool
	Content bool
	ContentSize bool
	ContentSha256 bool
	CreatedAt bool
	ShouldDelete bool
	MediaType bool
	IsBinary bool
}

// CurrentFileView is generated canonical view metadata for file.
type CurrentFileView struct {
	ReportId *string `sqlx:"report_id,primaryKey,refTable=report_versions,refColumn=report_id,required=true" validate:"required"`
	VersionNo *int `sqlx:"version_no,primaryKey,refTable=report_versions,refColumn=version_no,required=true"`
	ResourceId *string `sqlx:"resource_id,primaryKey,required=true" validate:"required"`
	Namespace *string `validate:"required" sqlx:"namespace,required=true"`
	ResourcePath *string `validate:"required" sqlx:"resource_path,required=true"`
	Content *string `validate:"required" sqlx:"content,required=true"`
	ContentSize *int `validate:"gte=0" sqlx:"content_size,required=true"`
	ContentSha256 *string `validate:"required" sqlx:"content_sha256,required=true"`
	CreatedAt *time.Time `validate:"required" sqlx:"created_at,required=true"`
	MediaType *string `sqlx:"media_type"`
	IsBinary *int `sqlx:"is_binary,required=true"`
}

type FileKeysRow struct {
	ReportId *string `sqlx:"report_id"`
	VersionNo *int `sqlx:"version_no"`
	ResourceId *string `sqlx:"resource_id"`
}
