package writer

import (
	json "encoding/json"
	time "time"
)

// ReportVersion is generated canonical view metadata for version.
type ReportVersion struct {
	ComponentSpecJson json.RawMessage `sqlx:"component_spec_json,enc=JSON,required=true" validate:"required"`
	DqlExportLimitsJson json.RawMessage `sqlx:"dql_export_limits_json,enc=JSON"`
	TypeManifestJson json.RawMessage `sqlx:"type_manifest_json,enc=JSON,required=true" validate:"required"`
	ResourceManifestJson json.RawMessage `sqlx:"resource_manifest_json,enc=JSON"`
	ComponentDescriptorJson json.RawMessage `sqlx:"component_descriptor_json,enc=JSON"`
	CompileDiagnosticsJson json.RawMessage `sqlx:"compile_diagnostics_json,enc=JSON"`
	ReportId *string `sqlx:"report_id,primaryKey,refTable=reports,refColumn=id,required=true" validate:"required"`
	VersionNo *int `sqlx:"version_no,primaryKey,required=true"`
	State *string `validate:"required,choice(draft,validated,published,superseded,failed)" sqlx:"state,required=true"`
	AuthoringMode *string `validate:"required,choice(sql,dql,structured)" sqlx:"authoring_mode,required=true"`
	SpecFormatVersion *string `validate:"required" sqlx:"spec_format_version,required=true"`
	SpecHash *string `validate:"required" sqlx:"spec_hash,required=true"`
	CompileStatus *string `validate:"required,choice(pending,valid,invalid,error)" sqlx:"compile_status,required=true"`
	DatlyVersion *string `validate:"required" sqlx:"datly_version,required=true"`
	CompilerVersion *string `validate:"required" sqlx:"compiler_version,required=true"`
	CreatedBy *string `validate:"required" sqlx:"created_by,required=true"`
	SourceRevision *int `writer:"concurrency" sqlx:"source_revision,required=true"`
	AuthoredSql *string `sqlx:"authored_sql"`
	AuthoredDql *string `sqlx:"authored_dql"`
	GeneratedDql *string `sqlx:"generated_dql"`
	Notes *string `sqlx:"notes"`
	CreatedAt *time.Time `sqlx:"created_at,required=true"`
	ValidatedAt *time.Time `sqlx:"validated_at"`
	PublishedAt *time.Time `sqlx:"published_at"`
	Has *ReportVersionHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"ReportVersionHas"`
}

type ReportVersionHas struct {
	ComponentSpecJson bool
	DqlExportLimitsJson bool
	TypeManifestJson bool
	ResourceManifestJson bool
	ComponentDescriptorJson bool
	CompileDiagnosticsJson bool
	ReportId bool
	VersionNo bool
	State bool
	AuthoringMode bool
	SpecFormatVersion bool
	SpecHash bool
	CompileStatus bool
	DatlyVersion bool
	CompilerVersion bool
	CreatedBy bool
	SourceRevision bool
	AuthoredSql bool
	AuthoredDql bool
	GeneratedDql bool
	Notes bool
	CreatedAt bool
	ValidatedAt bool
	PublishedAt bool
}

// CurrentVersionView is generated canonical view metadata for version.
type CurrentVersionView struct {
	ComponentSpecJson json.RawMessage `sqlx:"component_spec_json,enc=JSON,required=true" validate:"required"`
	DqlExportLimitsJson json.RawMessage `sqlx:"dql_export_limits_json,enc=JSON"`
	TypeManifestJson json.RawMessage `sqlx:"type_manifest_json,enc=JSON,required=true" validate:"required"`
	ResourceManifestJson json.RawMessage `sqlx:"resource_manifest_json,enc=JSON"`
	ComponentDescriptorJson json.RawMessage `sqlx:"component_descriptor_json,enc=JSON"`
	CompileDiagnosticsJson json.RawMessage `sqlx:"compile_diagnostics_json,enc=JSON"`
	ReportId *string `sqlx:"report_id,primaryKey,refTable=reports,refColumn=id,required=true" validate:"required"`
	VersionNo *int `sqlx:"version_no,primaryKey,required=true"`
	State *string `validate:"required,choice(draft,validated,published,superseded,failed)" sqlx:"state,required=true"`
	AuthoringMode *string `validate:"required,choice(sql,dql,structured)" sqlx:"authoring_mode,required=true"`
	SpecFormatVersion *string `validate:"required" sqlx:"spec_format_version,required=true"`
	SpecHash *string `validate:"required" sqlx:"spec_hash,required=true"`
	CompileStatus *string `validate:"required,choice(pending,valid,invalid,error)" sqlx:"compile_status,required=true"`
	DatlyVersion *string `validate:"required" sqlx:"datly_version,required=true"`
	CompilerVersion *string `validate:"required" sqlx:"compiler_version,required=true"`
	CreatedBy *string `validate:"required" sqlx:"created_by,required=true"`
	SourceRevision *int `sqlx:"source_revision,required=true"`
	AuthoredSql *string `sqlx:"authored_sql"`
	AuthoredDql *string `sqlx:"authored_dql"`
	GeneratedDql *string `sqlx:"generated_dql"`
	Notes *string `sqlx:"notes"`
	CreatedAt *time.Time `sqlx:"created_at,required=true"`
	ValidatedAt *time.Time `sqlx:"validated_at"`
	PublishedAt *time.Time `sqlx:"published_at"`
}

type VersionKeysRow struct {
	ReportId *string `sqlx:"report_id"`
	VersionNo *int `sqlx:"version_no"`
}
