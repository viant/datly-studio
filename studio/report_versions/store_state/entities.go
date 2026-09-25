package store_state

import (
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
func (entity *StoredVersion) GetPublishedAt() *time.Time {
	return entity.PublishedAt
}
func (entity *StoredVersion) SetPublishedAt(value *time.Time) {
	entity.PublishedAt = value
	if entity.Has == nil {
		entity.Has = &StoredVersionHas{}
	}
	entity.Has.PublishedAt = true
}
