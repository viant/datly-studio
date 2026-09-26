package get

import (
	json "encoding/json"
	time "time"
)

// VersionGetOutput is the generated output scaffold for version.
type VersionGetOutput struct {
	Item                *Version        `parameter:"Item,kind=output,in=view,dataType=*Version" json:"-" view:"version,type=Version,table=report_versions,limit=1" sql:"uri=studio_report_versions_get_version:sql/version.sql"`
	ResponseReportId    string          `parameter:"ResponseReportId,kind=output,in=body,dataType=string" json:"reportId"`
	ResponseVersionNo   int             `parameter:"ResponseVersionNo,kind=output,in=body,dataType=int" json:"versionNo"`
	State               string          `parameter:"State,kind=output,in=body,dataType=string" json:"state"`
	AuthoringMode       string          `parameter:"AuthoringMode,kind=output,in=body,dataType=string" json:"authoringMode"`
	AuthoredSql         string          `parameter:"AuthoredSql,kind=output,in=body,dataType=string" json:"authoredSql,omitempty"`
	AuthoredDql         string          `parameter:"AuthoredDql,kind=output,in=body,dataType=string" json:"authoredDql,omitempty"`
	ComponentSpec       json.RawMessage `parameter:"ComponentSpec,kind=output,in=body,dataType=json.RawMessage" json:"componentSpec,omitempty"`
	DqlExportLimits     json.RawMessage `parameter:"DqlExportLimits,kind=output,in=body,dataType=json.RawMessage" json:"dqlExportLimits,omitempty"`
	TypeManifest        json.RawMessage `parameter:"TypeManifest,kind=output,in=body,dataType=json.RawMessage" json:"typeManifest,omitempty"`
	ResourceManifest    json.RawMessage `parameter:"ResourceManifest,kind=output,in=body,dataType=json.RawMessage" json:"resourceManifest,omitempty"`
	ComponentDescriptor json.RawMessage `parameter:"ComponentDescriptor,kind=output,in=body,dataType=json.RawMessage" json:"componentDescriptor,omitempty"`
	SpecFormatVersion   string          `parameter:"SpecFormatVersion,kind=output,in=body,dataType=string" json:"specFormatVersion"`
	SpecHash            string          `parameter:"SpecHash,kind=output,in=body,dataType=string" json:"specHash"`
	GeneratedDql        string          `parameter:"GeneratedDql,kind=output,in=body,dataType=string" json:"generatedDql,omitempty"`
	CompileStatus       string          `parameter:"CompileStatus,kind=output,in=body,dataType=string" json:"compileStatus"`
	CompileDiagnostics  json.RawMessage `parameter:"CompileDiagnostics,kind=output,in=body,dataType=json.RawMessage" json:"compileDiagnostics,omitempty"`
	DatlyVersion        string          `parameter:"DatlyVersion,kind=output,in=body,dataType=string" json:"datlyVersion"`
	CompilerVersion     string          `parameter:"CompilerVersion,kind=output,in=body,dataType=string" json:"compilerVersion"`
	SourceRevision      int64           `parameter:"SourceRevision,kind=output,in=body,dataType=int64" json:"sourceRevision"`
	Notes               string          `parameter:"Notes,kind=output,in=body,dataType=string" json:"notes,omitempty"`
	CreatedBy           string          `parameter:"CreatedBy,kind=output,in=body,dataType=string" json:"createdBy"`
	CreatedAt           time.Time       `parameter:"CreatedAt,kind=output,in=body,dataType=time.Time" json:"createdAt"`
	ValidatedAt         *time.Time      `parameter:"ValidatedAt,kind=output,in=body,dataType=*time.Time" json:"validatedAt,omitempty"`
	PublishedAt         *time.Time      `parameter:"PublishedAt,kind=output,in=body,dataType=*time.Time" json:"publishedAt,omitempty"`
}
