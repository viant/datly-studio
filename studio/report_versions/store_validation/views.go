package store_validation

import (
	json "encoding/json"
	time "time"
)

// StoredVersion is generated canonical view metadata for version.
type StoredVersion struct {
	ReportId               string            `sqlx:"report_id,primaryKey,refTable=components,refColumn=id,required=true"`
	VersionNo              int               `sqlx:"version_no,primaryKey,required=true"`
	CompileStatus          string            `sqlx:"compile_status,required=true"`
	CompileDiagnosticsJson json.RawMessage   `sqlx:"compile_diagnostics_json,enc=JSON"`
	ValidatedAt            *time.Time        `sqlx:"validated_at"`
	SourceRevision         *int64            `writer:"concurrency" sqlx:"source_revision,required=true"`
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
	ReportId               string          `sqlx:"report_id,primaryKey,refTable=components,refColumn=id,required=true"`
	VersionNo              int             `sqlx:"version_no,primaryKey,required=true"`
	CompileStatus          string          `sqlx:"compile_status,required=true"`
	CompileDiagnosticsJson json.RawMessage `sqlx:"compile_diagnostics_json,enc=JSON"`
	ValidatedAt            *time.Time      `sqlx:"validated_at"`
	SourceRevision         *int64          `sqlx:"source_revision,required=true"`
}

type VersionKeysRow struct {
	ReportId  string `sqlx:"report_id"`
	VersionNo int    `sqlx:"version_no"`
}
