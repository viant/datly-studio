package store_edit

import (
	json "encoding/json"
)

// StoredVersion is generated canonical view metadata for version.
type StoredVersion struct {
	ReportId               string            `sqlx:"report_id,primaryKey"`
	VersionNo              int               `sqlx:"version_no,primaryKey"`
	State                  string            `sqlx:"state"`
	AuthoredSql            *string           `sqlx:"authored_sql"`
	AuthoredDql            *string           `sqlx:"authored_dql"`
	ComponentSpecJson      json.RawMessage   `sqlx:"component_spec_json,enc=JSON"`
	SpecHash               string            `sqlx:"spec_hash"`
	GeneratedDql           *string           `sqlx:"generated_dql"`
	CompileStatus          string            `sqlx:"compile_status"`
	CompileDiagnosticsJson json.RawMessage   `sqlx:"compile_diagnostics_json,enc=JSON"`
	SourceRevision         *int64            `writer:"concurrency" sqlx:"source_revision"`
	Has                    *StoredVersionHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredVersionHas"`
}

type StoredVersionHas struct {
	ReportId               bool
	VersionNo              bool
	State                  bool
	AuthoredSql            bool
	AuthoredDql            bool
	ComponentSpecJson      bool
	SpecHash               bool
	GeneratedDql           bool
	CompileStatus          bool
	CompileDiagnosticsJson bool
	SourceRevision         bool
}

// CurrentVersionView is generated canonical view metadata for version.
type CurrentVersionView struct {
	ReportId               string          `sqlx:"report_id,primaryKey"`
	VersionNo              int             `sqlx:"version_no,primaryKey"`
	State                  string          `sqlx:"state"`
	AuthoredSql            *string         `sqlx:"authored_sql"`
	AuthoredDql            *string         `sqlx:"authored_dql"`
	ComponentSpecJson      json.RawMessage `sqlx:"component_spec_json,enc=JSON"`
	SpecHash               string          `sqlx:"spec_hash"`
	GeneratedDql           *string         `sqlx:"generated_dql"`
	CompileStatus          string          `sqlx:"compile_status"`
	CompileDiagnosticsJson json.RawMessage `sqlx:"compile_diagnostics_json,enc=JSON"`
	SourceRevision         *int64          `sqlx:"source_revision"`
}

type VersionKeysRow struct {
	ReportId  string `sqlx:"report_id"`
	VersionNo int    `sqlx:"version_no"`
}
