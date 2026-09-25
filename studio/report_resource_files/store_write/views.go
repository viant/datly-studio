package store_write

import (
	time "time"
)

// StoredFile is generated canonical view metadata for file.
type StoredFile struct {
	ReportId      string         `sqlx:"report_id,primaryKey"`
	VersionNo     int            `sqlx:"version_no,primaryKey"`
	ResourceId    string         `sqlx:"resource_id,primaryKey"`
	Namespace     string         `sqlx:"namespace"`
	ResourcePath  string         `sqlx:"resource_path"`
	MediaType     *string        `sqlx:"media_type"`
	Content       []byte         `sqlx:"content"`
	ContentSize   int64          `sqlx:"content_size"`
	ContentSha256 string         `sqlx:"content_sha256"`
	IsBinary      bool           `sqlx:"is_binary"`
	CreatedAt     time.Time      `sqlx:"created_at"`
	ShouldDelete  bool           `sqlx:"-" writer:"delete"`
	Has           *StoredFileHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredFileHas"`
}

type StoredFileHas struct {
	ReportId      bool
	VersionNo     bool
	ResourceId    bool
	Namespace     bool
	ResourcePath  bool
	MediaType     bool
	Content       bool
	ContentSize   bool
	ContentSha256 bool
	IsBinary      bool
	CreatedAt     bool
	ShouldDelete  bool
}

// CurrentFileView is generated canonical view metadata for file.
type CurrentFileView struct {
	ReportId      string    `sqlx:"report_id,primaryKey"`
	VersionNo     int       `sqlx:"version_no,primaryKey"`
	ResourceId    string    `sqlx:"resource_id,primaryKey"`
	Namespace     string    `sqlx:"namespace"`
	ResourcePath  string    `sqlx:"resource_path"`
	MediaType     *string   `sqlx:"media_type"`
	Content       []byte    `sqlx:"content"`
	ContentSize   int64     `sqlx:"content_size"`
	ContentSha256 string    `sqlx:"content_sha256"`
	IsBinary      bool      `sqlx:"is_binary"`
	CreatedAt     time.Time `sqlx:"created_at"`
}

type FileKeysRow struct {
	ReportId   string `sqlx:"report_id"`
	VersionNo  int    `sqlx:"version_no"`
	ResourceId string `sqlx:"resource_id"`
}
