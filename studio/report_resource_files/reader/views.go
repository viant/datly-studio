package reader

import (
	time "time"
)

// ReportResourceFile is generated canonical view metadata for file.
type ReportResourceFile struct {
	ReportId *string `sqlx:"report_id"`
	VersionNo *int `sqlx:"version_no"`
	ResourceId *string `sqlx:"resource_id"`
	Namespace *string `sqlx:"namespace"`
	ResourcePath *string `sqlx:"resource_path"`
	MediaType *string `sqlx:"media_type"`
	Content *string `sqlx:"content"`
	ContentSize *int `sqlx:"content_size"`
	ContentSha256 *string `sqlx:"content_sha256"`
	IsBinary *int `sqlx:"is_binary"`
	CreatedAt *time.Time `sqlx:"created_at"`
}
