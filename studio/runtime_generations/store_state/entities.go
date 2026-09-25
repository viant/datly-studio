package store_state

import (
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
func (entity *StoredGeneration) GetReportCount() *int {
	return entity.ReportCount
}
func (entity *StoredGeneration) SetReportCount(value *int) {
	entity.ReportCount = value
	if entity.Has == nil {
		entity.Has = &StoredGenerationHas{}
	}
	entity.Has.ReportCount = true
}
func (entity *StoredGeneration) GetActivatedAt() *time.Time {
	return entity.ActivatedAt
}
func (entity *StoredGeneration) SetActivatedAt(value *time.Time) {
	entity.ActivatedAt = value
	if entity.Has == nil {
		entity.Has = &StoredGenerationHas{}
	}
	entity.Has.ActivatedAt = true
}
func (entity *StoredGeneration) GetRetiredAt() *time.Time {
	return entity.RetiredAt
}
func (entity *StoredGeneration) SetRetiredAt(value *time.Time) {
	entity.RetiredAt = value
	if entity.Has == nil {
		entity.Has = &StoredGenerationHas{}
	}
	entity.Has.RetiredAt = true
}
