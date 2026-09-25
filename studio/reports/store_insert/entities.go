package store_insert

import (
	time "time"
)

func (entity *StoredReport) GetId() string {
	return entity.Id
}
func (entity *StoredReport) SetId(value string) {
	entity.Id = value
	if entity.Has == nil {
		entity.Has = &StoredReportHas{}
	}
	entity.Has.Id = true
}
func (entity *StoredReport) GetNamespace() string {
	return entity.Namespace
}
func (entity *StoredReport) SetNamespace(value string) {
	entity.Namespace = value
	if entity.Has == nil {
		entity.Has = &StoredReportHas{}
	}
	entity.Has.Namespace = true
}
func (entity *StoredReport) GetSlug() string {
	return entity.Slug
}
func (entity *StoredReport) SetSlug(value string) {
	entity.Slug = value
	if entity.Has == nil {
		entity.Has = &StoredReportHas{}
	}
	entity.Has.Slug = true
}
func (entity *StoredReport) GetTitle() string {
	return entity.Title
}
func (entity *StoredReport) SetTitle(value string) {
	entity.Title = value
	if entity.Has == nil {
		entity.Has = &StoredReportHas{}
	}
	entity.Has.Title = true
}
func (entity *StoredReport) GetDescription() *string {
	return entity.Description
}
func (entity *StoredReport) SetDescription(value *string) {
	entity.Description = value
	if entity.Has == nil {
		entity.Has = &StoredReportHas{}
	}
	entity.Has.Description = true
}
func (entity *StoredReport) GetOwnerId() string {
	return entity.OwnerId
}
func (entity *StoredReport) SetOwnerId(value string) {
	entity.OwnerId = value
	if entity.Has == nil {
		entity.Has = &StoredReportHas{}
	}
	entity.Has.OwnerId = true
}
func (entity *StoredReport) GetStatus() string {
	return entity.Status
}
func (entity *StoredReport) SetStatus(value string) {
	entity.Status = value
	if entity.Has == nil {
		entity.Has = &StoredReportHas{}
	}
	entity.Has.Status = true
}
func (entity *StoredReport) GetDefaultConnectorName() string {
	return entity.DefaultConnectorName
}
func (entity *StoredReport) SetDefaultConnectorName(value string) {
	entity.DefaultConnectorName = value
	if entity.Has == nil {
		entity.Has = &StoredReportHas{}
	}
	entity.Has.DefaultConnectorName = true
}
func (entity *StoredReport) GetComponentScope() string {
	return entity.ComponentScope
}
func (entity *StoredReport) SetComponentScope(value string) {
	entity.ComponentScope = value
	if entity.Has == nil {
		entity.Has = &StoredReportHas{}
	}
	entity.Has.ComponentScope = true
}
func (entity *StoredReport) GetComponentName() string {
	return entity.ComponentName
}
func (entity *StoredReport) SetComponentName(value string) {
	entity.ComponentName = value
	if entity.Has == nil {
		entity.Has = &StoredReportHas{}
	}
	entity.Has.ComponentName = true
}
func (entity *StoredReport) GetEtag() int64 {
	return entity.Etag
}
func (entity *StoredReport) SetEtag(value int64) {
	entity.Etag = value
	if entity.Has == nil {
		entity.Has = &StoredReportHas{}
	}
	entity.Has.Etag = true
}
func (entity *StoredReport) GetCreatedAt() time.Time {
	return entity.CreatedAt
}
func (entity *StoredReport) SetCreatedAt(value time.Time) {
	entity.CreatedAt = value
	if entity.Has == nil {
		entity.Has = &StoredReportHas{}
	}
	entity.Has.CreatedAt = true
}
func (entity *StoredReport) GetUpdatedAt() time.Time {
	return entity.UpdatedAt
}
func (entity *StoredReport) SetUpdatedAt(value time.Time) {
	entity.UpdatedAt = value
	if entity.Has == nil {
		entity.Has = &StoredReportHas{}
	}
	entity.Has.UpdatedAt = true
}
