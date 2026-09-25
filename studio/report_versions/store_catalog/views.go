package store_catalog

import (
	json "encoding/json"
	time "time"
)

// StoredVersion is generated canonical view metadata for version.
type StoredVersion struct {
	ReportId                string          `sqlx:"report_id"`
	VersionNo               int             `sqlx:"version_no"`
	State                   string          `sqlx:"state"`
	AuthoringMode           string          `sqlx:"authoring_mode"`
	AuthoredSql             *string         `sqlx:"authored_sql"`
	AuthoredDql             *string         `sqlx:"authored_dql"`
	ComponentSpecJson       json.RawMessage `sqlx:"component_spec_json,enc=JSON"`
	SpecFormatVersion       string          `sqlx:"spec_format_version"`
	SpecHash                string          `sqlx:"spec_hash"`
	GeneratedDql            *string         `sqlx:"generated_dql"`
	DqlExportLimitsJson     json.RawMessage `sqlx:"dql_export_limits_json,enc=JSON"`
	TypeManifestJson        json.RawMessage `sqlx:"type_manifest_json,enc=JSON"`
	ResourceManifestJson    json.RawMessage `sqlx:"resource_manifest_json,enc=JSON"`
	ComponentDescriptorJson json.RawMessage `sqlx:"component_descriptor_json,enc=JSON"`
	CompileStatus           string          `sqlx:"compile_status"`
	CompileDiagnosticsJson  json.RawMessage `sqlx:"compile_diagnostics_json,enc=JSON"`
	DatlyVersion            string          `sqlx:"datly_version"`
	CompilerVersion         string          `sqlx:"compiler_version"`
	SourceRevision          int64           `sqlx:"source_revision"`
	Notes                   *string         `sqlx:"notes"`
	CreatedBy               string          `sqlx:"created_by"`
	CreatedAt               time.Time       `sqlx:"created_at"`
	ValidatedAt             *time.Time      `sqlx:"validated_at"`
	PublishedAt             *time.Time      `sqlx:"published_at"`
}
