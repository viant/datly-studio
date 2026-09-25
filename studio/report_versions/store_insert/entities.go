package store_insert

import (
	json "encoding/json"
	time "time"
)

func (entity *StoredVersion) GetReportId() string {
	return entity.ReportId
}
func (entity *StoredVersion) SetReportId(value string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.ReportId = true
}
func (entity *StoredVersion) GetVersionNo() int {
	return entity.VersionNo
}
func (entity *StoredVersion) SetVersionNo(value int) {
	entity.VersionNo = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.VersionNo = true
}
func (entity *StoredVersion) GetState() string {
	return entity.State
}
func (entity *StoredVersion) SetState(value string) {
	entity.State = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.State = true
}
func (entity *StoredVersion) GetAuthoringMode() string {
	return entity.AuthoringMode
}
func (entity *StoredVersion) SetAuthoringMode(value string) {
	entity.AuthoringMode = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.AuthoringMode = true
}
func (entity *StoredVersion) GetAuthoredSql() *string {
	return entity.AuthoredSql
}
func (entity *StoredVersion) SetAuthoredSql(value *string) {
	entity.AuthoredSql = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.AuthoredSql = true
}
func (entity *StoredVersion) GetAuthoredDql() *string {
	return entity.AuthoredDql
}
func (entity *StoredVersion) SetAuthoredDql(value *string) {
	entity.AuthoredDql = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.AuthoredDql = true
}
func (entity *StoredVersion) GetComponentSpecJson() json.RawMessage {
	return entity.ComponentSpecJson
}
func (entity *StoredVersion) SetComponentSpecJson(value json.RawMessage) {
	entity.ComponentSpecJson = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.ComponentSpecJson = true
}
func (entity *StoredVersion) GetSpecFormatVersion() string {
	return entity.SpecFormatVersion
}
func (entity *StoredVersion) SetSpecFormatVersion(value string) {
	entity.SpecFormatVersion = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.SpecFormatVersion = true
}
func (entity *StoredVersion) GetSpecHash() string {
	return entity.SpecHash
}
func (entity *StoredVersion) SetSpecHash(value string) {
	entity.SpecHash = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.SpecHash = true
}
func (entity *StoredVersion) GetGeneratedDql() *string {
	return entity.GeneratedDql
}
func (entity *StoredVersion) SetGeneratedDql(value *string) {
	entity.GeneratedDql = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.GeneratedDql = true
}
func (entity *StoredVersion) GetTypeManifestJson() json.RawMessage {
	return entity.TypeManifestJson
}
func (entity *StoredVersion) SetTypeManifestJson(value json.RawMessage) {
	entity.TypeManifestJson = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.TypeManifestJson = true
}
func (entity *StoredVersion) GetCompileStatus() string {
	return entity.CompileStatus
}
func (entity *StoredVersion) SetCompileStatus(value string) {
	entity.CompileStatus = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.CompileStatus = true
}
func (entity *StoredVersion) GetDatlyVersion() string {
	return entity.DatlyVersion
}
func (entity *StoredVersion) SetDatlyVersion(value string) {
	entity.DatlyVersion = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.DatlyVersion = true
}
func (entity *StoredVersion) GetCompilerVersion() string {
	return entity.CompilerVersion
}
func (entity *StoredVersion) SetCompilerVersion(value string) {
	entity.CompilerVersion = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.CompilerVersion = true
}
func (entity *StoredVersion) GetSourceRevision() int64 {
	return entity.SourceRevision
}
func (entity *StoredVersion) SetSourceRevision(value int64) {
	entity.SourceRevision = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.SourceRevision = true
}
func (entity *StoredVersion) GetNotes() *string {
	return entity.Notes
}
func (entity *StoredVersion) SetNotes(value *string) {
	entity.Notes = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.Notes = true
}
func (entity *StoredVersion) GetCreatedBy() string {
	return entity.CreatedBy
}
func (entity *StoredVersion) SetCreatedBy(value string) {
	entity.CreatedBy = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.CreatedBy = true
}
func (entity *StoredVersion) GetCreatedAt() time.Time {
	return entity.CreatedAt
}
func (entity *StoredVersion) SetCreatedAt(value time.Time) {
	entity.CreatedAt = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.CreatedAt = true
}
