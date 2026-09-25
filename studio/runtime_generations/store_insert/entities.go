package store_insert

import (
	json "encoding/json"
	time "time"
)

func (entity *StoredGeneration) GetGenerationNo() int64 {
	return entity.GenerationNo
}
func (entity *StoredGeneration) SetGenerationNo(value int64) {
	entity.GenerationNo = value
	if entity.Has == nil {
		entity.Has = &StoredGenerationHas{}
	}
	entity.Has.GenerationNo = true
}
func (entity *StoredGeneration) GetSourceRevision() string {
	return entity.SourceRevision
}
func (entity *StoredGeneration) SetSourceRevision(value string) {
	entity.SourceRevision = value
	if entity.Has == nil {
		entity.Has = &StoredGenerationHas{}
	}
	entity.Has.SourceRevision = true
}
func (entity *StoredGeneration) GetStatus() string {
	return entity.Status
}
func (entity *StoredGeneration) SetStatus(value string) {
	entity.Status = value
	if entity.Has == nil {
		entity.Has = &StoredGenerationHas{}
	}
	entity.Has.Status = true
}
func (entity *StoredGeneration) GetReportCount() int {
	return entity.ReportCount
}
func (entity *StoredGeneration) SetReportCount(value int) {
	entity.ReportCount = value
	if entity.Has == nil {
		entity.Has = &StoredGenerationHas{}
	}
	entity.Has.ReportCount = true
}
func (entity *StoredGeneration) GetBuildManifestJson() json.RawMessage {
	return entity.BuildManifestJson
}
func (entity *StoredGeneration) SetBuildManifestJson(value json.RawMessage) {
	entity.BuildManifestJson = value
	if entity.Has == nil {
		entity.Has = &StoredGenerationHas{}
	}
	entity.Has.BuildManifestJson = true
}
func (entity *StoredGeneration) GetRequestedBy() string {
	return entity.RequestedBy
}
func (entity *StoredGeneration) SetRequestedBy(value string) {
	entity.RequestedBy = value
	if entity.Has == nil {
		entity.Has = &StoredGenerationHas{}
	}
	entity.Has.RequestedBy = true
}
func (entity *StoredGeneration) GetRequestedAt() time.Time {
	return entity.RequestedAt
}
func (entity *StoredGeneration) SetRequestedAt(value time.Time) {
	entity.RequestedAt = value
	if entity.Has == nil {
		entity.Has = &StoredGenerationHas{}
	}
	entity.Has.RequestedAt = true
}
