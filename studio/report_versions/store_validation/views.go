package store_validation

import (
	json "encoding/json"
	time "time"
)

// StoredVersion is generated canonical view metadata for version.
type StoredVersion struct {
	ReportId               string            `sqlx:"report_id,primaryKey"`
	VersionNo              int               `sqlx:"version_no,primaryKey"`
	CompileStatus          string            `sqlx:"compile_status"`
	CompileDiagnosticsJson json.RawMessage   `sqlx:"compile_diagnostics_json,enc=JSON"`
	ValidatedAt            *time.Time        `sqlx:"validated_at"`
	SourceRevision         *int64            `writer:"concurrency" sqlx:"source_revision"`
	Has                    *StoredVersionHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredVersionHas"`
}

type StoredVersionHas struct {
	ReportId               bool
	VersionNo              bool
	CompileStatus          bool
	CompileDiagnosticsJson bool
	ValidatedAt            bool
	SourceRevision         bool
}

// CurrentVersionView is generated canonical view metadata for version.
type CurrentVersionView struct {
	ReportId               string          `sqlx:"report_id,primaryKey"`
	VersionNo              int             `sqlx:"version_no,primaryKey"`
	CompileStatus          string          `sqlx:"compile_status"`
	CompileDiagnosticsJson json.RawMessage `sqlx:"compile_diagnostics_json,enc=JSON"`
	ValidatedAt            *time.Time      `sqlx:"validated_at"`
	SourceRevision         *int64          `sqlx:"source_revision"`
}

type VersionKeysRow struct {
	ReportId  string `sqlx:"report_id"`
	VersionNo int    `sqlx:"version_no"`
}
