package store_recover

import (
	time "time"
)

func (entity *StoredPublication) GetReportId() string {
	return entity.ReportId
}
func (entity *StoredPublication) SetReportId(value string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &StoredPublicationHas{}
	}
	entity.Has.ReportId = true
}
func (entity *StoredPublication) GetActiveVersionNo() int {
	return entity.ActiveVersionNo
}
func (entity *StoredPublication) SetActiveVersionNo(value int) {
	entity.ActiveVersionNo = value
	if entity.Has == nil {
		entity.Has = &StoredPublicationHas{}
	}
	entity.Has.ActiveVersionNo = true
}
func (entity *StoredPublication) GetDesiredVersionNo() *int {
	return entity.DesiredVersionNo
}
func (entity *StoredPublication) SetDesiredVersionNo(value *int) {
	entity.DesiredVersionNo = value
	if entity.Has == nil {
		entity.Has = &StoredPublicationHas{}
	}
	entity.Has.DesiredVersionNo = true
}
func (entity *StoredPublication) GetDesiredGeneration() *int64 {
	return entity.DesiredGeneration
}
func (entity *StoredPublication) SetDesiredGeneration(value *int64) {
	entity.DesiredGeneration = value
	if entity.Has == nil {
		entity.Has = &StoredPublicationHas{}
	}
	entity.Has.DesiredGeneration = true
}
func (entity *StoredPublication) GetActiveGeneration() *int64 {
	return entity.ActiveGeneration
}
func (entity *StoredPublication) SetActiveGeneration(value *int64) {
	entity.ActiveGeneration = value
	if entity.Has == nil {
		entity.Has = &StoredPublicationHas{}
	}
	entity.Has.ActiveGeneration = true
}
func (entity *StoredPublication) GetPublicationStatus() string {
	return entity.PublicationStatus
}
func (entity *StoredPublication) SetPublicationStatus(value string) {
	entity.PublicationStatus = value
	if entity.Has == nil {
		entity.Has = &StoredPublicationHas{}
	}
	entity.Has.PublicationStatus = true
}
func (entity *StoredPublication) GetRuntimeRevision() *string {
	return entity.RuntimeRevision
}
func (entity *StoredPublication) SetRuntimeRevision(value *string) {
	entity.RuntimeRevision = value
	if entity.Has == nil {
		entity.Has = &StoredPublicationHas{}
	}
	entity.Has.RuntimeRevision = true
}
func (entity *StoredPublication) GetSpecHash() string {
	return entity.SpecHash
}
func (entity *StoredPublication) SetSpecHash(value string) {
	entity.SpecHash = value
	if entity.Has == nil {
		entity.Has = &StoredPublicationHas{}
	}
	entity.Has.SpecHash = true
}
func (entity *StoredPublication) GetPublishedBy() string {
	return entity.PublishedBy
}
func (entity *StoredPublication) SetPublishedBy(value string) {
	entity.PublishedBy = value
	if entity.Has == nil {
		entity.Has = &StoredPublicationHas{}
	}
	entity.Has.PublishedBy = true
}
func (entity *StoredPublication) GetPublishedAt() *time.Time {
	return entity.PublishedAt
}
func (entity *StoredPublication) SetPublishedAt(value *time.Time) {
	entity.PublishedAt = value
	if entity.Has == nil {
		entity.Has = &StoredPublicationHas{}
	}
	entity.Has.PublishedAt = true
}
func (entity *StoredPublication) GetActivatedAt() *time.Time {
	return entity.ActivatedAt
}
func (entity *StoredPublication) SetActivatedAt(value *time.Time) {
	entity.ActivatedAt = value
	if entity.Has == nil {
		entity.Has = &StoredPublicationHas{}
	}
	entity.Has.ActivatedAt = true
}
func (entity *StoredPublication) GetFailureJson() *string {
	return entity.FailureJson
}
func (entity *StoredPublication) SetFailureJson(value *string) {
	entity.FailureJson = value
	if entity.Has == nil {
		entity.Has = &StoredPublicationHas{}
	}
	entity.Has.FailureJson = true
}
