package writer

import (
	json "encoding/json"
	time "time"
)

func (entity *ReportVersion) GetComponentSpecJson() json.RawMessage {
	return entity.ComponentSpecJson
}
func (entity *ReportVersion) SetComponentSpecJson(value json.RawMessage) {
	entity.ComponentSpecJson = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.ComponentSpecJson = true
}
func (entity *ReportVersion) GetDqlExportLimitsJson() json.RawMessage {
	return entity.DqlExportLimitsJson
}
func (entity *ReportVersion) SetDqlExportLimitsJson(value json.RawMessage) {
	entity.DqlExportLimitsJson = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.DqlExportLimitsJson = true
}
func (entity *ReportVersion) GetTypeManifestJson() json.RawMessage {
	return entity.TypeManifestJson
}
func (entity *ReportVersion) SetTypeManifestJson(value json.RawMessage) {
	entity.TypeManifestJson = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.TypeManifestJson = true
}
func (entity *ReportVersion) GetResourceManifestJson() json.RawMessage {
	return entity.ResourceManifestJson
}
func (entity *ReportVersion) SetResourceManifestJson(value json.RawMessage) {
	entity.ResourceManifestJson = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.ResourceManifestJson = true
}
func (entity *ReportVersion) GetComponentDescriptorJson() json.RawMessage {
	return entity.ComponentDescriptorJson
}
func (entity *ReportVersion) SetComponentDescriptorJson(value json.RawMessage) {
	entity.ComponentDescriptorJson = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.ComponentDescriptorJson = true
}
func (entity *ReportVersion) GetCompileDiagnosticsJson() json.RawMessage {
	return entity.CompileDiagnosticsJson
}
func (entity *ReportVersion) SetCompileDiagnosticsJson(value json.RawMessage) {
	entity.CompileDiagnosticsJson = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.CompileDiagnosticsJson = true
}
func (entity *ReportVersion) GetReportId() *string {
	return entity.ReportId
}
func (entity *ReportVersion) SetReportId(value *string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.ReportId = true
}
func (entity *ReportVersion) GetVersionNo() *int {
	return entity.VersionNo
}
func (entity *ReportVersion) SetVersionNo(value *int) {
	entity.VersionNo = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.VersionNo = true
}
func (entity *ReportVersion) GetState() *string {
	return entity.State
}
func (entity *ReportVersion) SetState(value *string) {
	entity.State = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.State = true
}
func (entity *ReportVersion) GetAuthoringMode() *string {
	return entity.AuthoringMode
}
func (entity *ReportVersion) SetAuthoringMode(value *string) {
	entity.AuthoringMode = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.AuthoringMode = true
}
func (entity *ReportVersion) GetSpecFormatVersion() *string {
	return entity.SpecFormatVersion
}
func (entity *ReportVersion) SetSpecFormatVersion(value *string) {
	entity.SpecFormatVersion = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.SpecFormatVersion = true
}
func (entity *ReportVersion) GetSpecHash() *string {
	return entity.SpecHash
}
func (entity *ReportVersion) SetSpecHash(value *string) {
	entity.SpecHash = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.SpecHash = true
}
func (entity *ReportVersion) GetCompileStatus() *string {
	return entity.CompileStatus
}
func (entity *ReportVersion) SetCompileStatus(value *string) {
	entity.CompileStatus = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.CompileStatus = true
}
func (entity *ReportVersion) GetDatlyVersion() *string {
	return entity.DatlyVersion
}
func (entity *ReportVersion) SetDatlyVersion(value *string) {
	entity.DatlyVersion = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.DatlyVersion = true
}
func (entity *ReportVersion) GetCompilerVersion() *string {
	return entity.CompilerVersion
}
func (entity *ReportVersion) SetCompilerVersion(value *string) {
	entity.CompilerVersion = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.CompilerVersion = true
}
func (entity *ReportVersion) GetCreatedBy() *string {
	return entity.CreatedBy
}
func (entity *ReportVersion) SetCreatedBy(value *string) {
	entity.CreatedBy = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.CreatedBy = true
}
func (entity *ReportVersion) GetSourceRevision() *int {
	return entity.SourceRevision
}
func (entity *ReportVersion) SetSourceRevision(value *int) {
	entity.SourceRevision = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.SourceRevision = true
}
func (entity *ReportVersion) GetAuthoredSql() *string {
	return entity.AuthoredSql
}
func (entity *ReportVersion) SetAuthoredSql(value *string) {
	entity.AuthoredSql = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.AuthoredSql = true
}
func (entity *ReportVersion) GetAuthoredDql() *string {
	return entity.AuthoredDql
}
func (entity *ReportVersion) SetAuthoredDql(value *string) {
	entity.AuthoredDql = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.AuthoredDql = true
}
func (entity *ReportVersion) GetGeneratedDql() *string {
	return entity.GeneratedDql
}
func (entity *ReportVersion) SetGeneratedDql(value *string) {
	entity.GeneratedDql = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.GeneratedDql = true
}
func (entity *ReportVersion) GetNotes() *string {
	return entity.Notes
}
func (entity *ReportVersion) SetNotes(value *string) {
	entity.Notes = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.Notes = true
}
func (entity *ReportVersion) GetCreatedAt() *time.Time {
	return entity.CreatedAt
}
func (entity *ReportVersion) SetCreatedAt(value *time.Time) {
	entity.CreatedAt = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.CreatedAt = true
}
func (entity *ReportVersion) GetValidatedAt() *time.Time {
	return entity.ValidatedAt
}
func (entity *ReportVersion) SetValidatedAt(value *time.Time) {
	entity.ValidatedAt = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.ValidatedAt = true
}
func (entity *ReportVersion) GetPublishedAt() *time.Time {
	return entity.PublishedAt
}
func (entity *ReportVersion) SetPublishedAt(value *time.Time) {
	entity.PublishedAt = value
	if entity.Has == nil {
		entity.Has = &ReportVersionHas{}
	}
	entity.Has.PublishedAt = true
}
