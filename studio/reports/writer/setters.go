package writer

import (
	time "time"
)

func (entity *Report) GetNamespace() string {
	return entity.Namespace
}
func (entity *Report) SetNamespace(value string) {
	entity.Namespace = value
	if entity.Has == nil {
		entity.Has = &ReportHas{}
	}
	entity.Has.Namespace = true
}
func (entity *Report) GetId() *string {
	return entity.Id
}
func (entity *Report) SetId(value *string) {
	entity.Id = value
	if entity.Has == nil {
		entity.Has = &ReportHas{}
	}
	entity.Has.Id = true
}
func (entity *Report) GetSlug() *string {
	return entity.Slug
}
func (entity *Report) SetSlug(value *string) {
	entity.Slug = value
	if entity.Has == nil {
		entity.Has = &ReportHas{}
	}
	entity.Has.Slug = true
}
func (entity *Report) GetTitle() *string {
	return entity.Title
}
func (entity *Report) SetTitle(value *string) {
	entity.Title = value
	if entity.Has == nil {
		entity.Has = &ReportHas{}
	}
	entity.Has.Title = true
}
func (entity *Report) GetOwnerId() *string {
	return entity.OwnerId
}
func (entity *Report) SetOwnerId(value *string) {
	entity.OwnerId = value
	if entity.Has == nil {
		entity.Has = &ReportHas{}
	}
	entity.Has.OwnerId = true
}
func (entity *Report) GetStatus() *string {
	return entity.Status
}
func (entity *Report) SetStatus(value *string) {
	entity.Status = value
	if entity.Has == nil {
		entity.Has = &ReportHas{}
	}
	entity.Has.Status = true
}
func (entity *Report) GetDefaultConnectorName() *string {
	return entity.DefaultConnectorName
}
func (entity *Report) SetDefaultConnectorName(value *string) {
	entity.DefaultConnectorName = value
	if entity.Has == nil {
		entity.Has = &ReportHas{}
	}
	entity.Has.DefaultConnectorName = true
}
func (entity *Report) GetComponentScope() *string {
	return entity.ComponentScope
}
func (entity *Report) SetComponentScope(value *string) {
	entity.ComponentScope = value
	if entity.Has == nil {
		entity.Has = &ReportHas{}
	}
	entity.Has.ComponentScope = true
}
func (entity *Report) GetComponentName() *string {
	return entity.ComponentName
}
func (entity *Report) SetComponentName(value *string) {
	entity.ComponentName = value
	if entity.Has == nil {
		entity.Has = &ReportHas{}
	}
	entity.Has.ComponentName = true
}
func (entity *Report) GetShouldDelete() bool {
	return entity.ShouldDelete
}
func (entity *Report) GetEtag() *int {
	return entity.Etag
}
func (entity *Report) SetEtag(value *int) {
	entity.Etag = value
	if entity.Has == nil {
		entity.Has = &ReportHas{}
	}
	entity.Has.Etag = true
}
func (entity *Report) GetDescription() *string {
	return entity.Description
}
func (entity *Report) SetDescription(value *string) {
	entity.Description = value
	if entity.Has == nil {
		entity.Has = &ReportHas{}
	}
	entity.Has.Description = true
}
func (entity *Report) GetCurrentDraftVersion() *int {
	return entity.CurrentDraftVersion
}
func (entity *Report) SetCurrentDraftVersion(value *int) {
	entity.CurrentDraftVersion = value
	if entity.Has == nil {
		entity.Has = &ReportHas{}
	}
	entity.Has.CurrentDraftVersion = true
}
func (entity *Report) GetCreatedAt() *time.Time {
	return entity.CreatedAt
}
func (entity *Report) SetCreatedAt(value *time.Time) {
	entity.CreatedAt = value
	if entity.Has == nil {
		entity.Has = &ReportHas{}
	}
	entity.Has.CreatedAt = true
}
func (entity *Report) GetUpdatedAt() *time.Time {
	return entity.UpdatedAt
}
func (entity *Report) SetUpdatedAt(value *time.Time) {
	entity.UpdatedAt = value
	if entity.Has == nil {
		entity.Has = &ReportHas{}
	}
	entity.Has.UpdatedAt = true
}
func (entity *Report) GetDeletedAt() *time.Time {
	return entity.DeletedAt
}
func (entity *Report) SetDeletedAt(value *time.Time) {
	entity.DeletedAt = value
	if entity.Has == nil {
		entity.Has = &ReportHas{}
	}
	entity.Has.DeletedAt = true
}
