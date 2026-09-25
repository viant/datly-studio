package store_edit

import (
	json "encoding/json"
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
func (entity *StoredVersion) GetCompileDiagnosticsJson() json.RawMessage {
	return entity.CompileDiagnosticsJson
}
func (entity *StoredVersion) SetCompileDiagnosticsJson(value json.RawMessage) {
	entity.CompileDiagnosticsJson = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.CompileDiagnosticsJson = true
}
func (entity *StoredVersion) GetSourceRevision() *int64 {
	return entity.SourceRevision
}
func (entity *StoredVersion) SetSourceRevision(value *int64) {
	entity.SourceRevision = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.SourceRevision = true
}
