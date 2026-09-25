package store_touch

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
func (entity *StoredVersion) GetValidatedAt() *time.Time {
	return entity.ValidatedAt
}
func (entity *StoredVersion) SetValidatedAt(value *time.Time) {
	entity.ValidatedAt = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.ValidatedAt = true
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
