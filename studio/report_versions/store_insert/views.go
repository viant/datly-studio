package store_insert

import (
	json "encoding/json"
	time "time"
)

// StoredVersion is generated canonical view metadata for version.
type StoredVersion struct {
	ReportId          string            `sqlx:"report_id,primaryKey"`
	VersionNo         int               `sqlx:"version_no,primaryKey"`
	State             string            `sqlx:"state"`
	AuthoringMode     string            `sqlx:"authoring_mode"`
	AuthoredSql       *string           `sqlx:"authored_sql"`
	AuthoredDql       *string           `sqlx:"authored_dql"`
	ComponentSpecJson json.RawMessage   `sqlx:"component_spec_json,enc=JSON"`
	SpecFormatVersion string            `sqlx:"spec_format_version"`
	SpecHash          string            `sqlx:"spec_hash"`
	GeneratedDql      *string           `sqlx:"generated_dql"`
	TypeManifestJson  json.RawMessage   `sqlx:"type_manifest_json,enc=JSON"`
	CompileStatus     string            `sqlx:"compile_status"`
	DatlyVersion      string            `sqlx:"datly_version"`
	CompilerVersion   string            `sqlx:"compiler_version"`
	SourceRevision    int64             `sqlx:"source_revision"`
	Notes             *string           `sqlx:"notes"`
	CreatedBy         string            `sqlx:"created_by"`
	CreatedAt         time.Time         `sqlx:"created_at"`
	Has               *StoredVersionHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"StoredVersionHas"`
}

type StoredVersionHas struct {
	ReportId          bool
	VersionNo         bool
	State             bool
	AuthoringMode     bool
	AuthoredSql       bool
	AuthoredDql       bool
	ComponentSpecJson bool
	SpecFormatVersion bool
	SpecHash          bool
	GeneratedDql      bool
	TypeManifestJson  bool
	CompileStatus     bool
	DatlyVersion      bool
	CompilerVersion   bool
	SourceRevision    bool
	Notes             bool
	CreatedBy         bool
	CreatedAt         bool
}
