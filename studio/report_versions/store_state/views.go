package store_state

import (
	time "time"
)

// StoredVersion is generated canonical view metadata for version.
type StoredVersion struct {
	ReportId    string            `sqlx:"report_id,primaryKey"`
	VersionNo   int               `sqlx:"version_no,primaryKey"`
	State       string            `writer:"concurrency" sqlx:"state"`
	PublishedAt *time.Time        `sqlx:"published_at"`
	Has         *StoredVersionHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredVersionHas"`
}

type StoredVersionHas struct {
	ReportId    bool
	VersionNo   bool
	State       bool
	PublishedAt bool
}

// CurrentVersionView is generated canonical view metadata for version.
type CurrentVersionView struct {
	ReportId    string     `sqlx:"report_id,primaryKey"`
	VersionNo   int        `sqlx:"version_no,primaryKey"`
	State       string     `sqlx:"state"`
	PublishedAt *time.Time `sqlx:"published_at"`
}

type VersionKeysRow struct {
	ReportId  string `sqlx:"report_id"`
	VersionNo int    `sqlx:"version_no"`
}
