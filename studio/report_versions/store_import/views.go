package store_import

import (
	json "encoding/json"
	time "time"
)

// ImportedVersion is generated canonical view metadata for version.
type ImportedVersion struct {
	ReportId          string                  `sqlx:"report_id,primaryKey"`
	VersionNo         int                     `sqlx:"version_no,primaryKey"`
	State             string                  `sqlx:"state"`
	AuthoringMode     string                  `sqlx:"authoring_mode"`
	AuthoredDql       *string                 `sqlx:"authored_dql"`
	GeneratedDql      *string                 `sqlx:"generated_dql"`
	ComponentSpecJson json.RawMessage         `sqlx:"component_spec_json,enc=JSON"`
	SpecFormatVersion string                  `sqlx:"spec_format_version"`
	SpecHash          string                  `sqlx:"spec_hash"`
	TypeManifestJson  json.RawMessage         `sqlx:"type_manifest_json,enc=JSON"`
	CompileStatus     string                  `sqlx:"compile_status"`
	DatlyVersion      string                  `sqlx:"datly_version"`
	CompilerVersion   string                  `sqlx:"compiler_version"`
	SourceRevision    int64                   `sqlx:"source_revision"`
	Notes             *string                 `sqlx:"notes"`
	CreatedBy         string                  `sqlx:"created_by"`
	CreatedAt         time.Time               `sqlx:"created_at"`
	File              []*ImportedResourceFile `view:"file,type=ImportedResourceFile,table=report_resource_files" on:"ReportId:version.report_id=ReportId:file.report_id,VersionNo:version.version_no=VersionNo:file.version_no" json:"file" sql:"uri=studio_report_versions_store_import_version:sql/file.sql"`
	Has               *ImportedVersionHas     `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"ImportedVersionHas"`
}

type ImportedVersionHas struct {
	ReportId          bool
	VersionNo         bool
	State             bool
	AuthoringMode     bool
	AuthoredDql       bool
	GeneratedDql      bool
	ComponentSpecJson bool
	SpecFormatVersion bool
	SpecHash          bool
	TypeManifestJson  bool
	CompileStatus     bool
	DatlyVersion      bool
	CompilerVersion   bool
	SourceRevision    bool
	Notes             bool
	CreatedBy         bool
	CreatedAt         bool
	File              bool
}

// ImportedResourceFile is generated canonical view metadata for version.
type ImportedResourceFile struct {
	ReportId      string                   `sqlx:"report_id,primaryKey"`
	VersionNo     int                      `sqlx:"version_no,primaryKey"`
	ResourceId    string                   `sqlx:"resource_id,primaryKey"`
	Namespace     string                   `sqlx:"namespace"`
	ResourcePath  string                   `sqlx:"resource_path"`
	Content       []byte                   `sqlx:"content"`
	ContentSize   int64                    `sqlx:"content_size"`
	ContentSha256 string                   `sqlx:"content_sha256"`
	IsBinary      bool                     `sqlx:"is_binary"`
	CreatedAt     time.Time                `sqlx:"created_at"`
	Has           *ImportedResourceFileHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"ImportedResourceFileHas"`
}

type ImportedResourceFileHas struct {
	ReportId      bool
	VersionNo     bool
	ResourceId    bool
	Namespace     bool
	ResourcePath  bool
	Content       bool
	ContentSize   bool
	ContentSha256 bool
	IsBinary      bool
	CreatedAt     bool
}
