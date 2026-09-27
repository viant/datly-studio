package create

import (
	time "time"
)

func (entity *NamespaceRecord) GetOwnerId() string {
	return entity.OwnerId
}
func (entity *NamespaceRecord) SetOwnerId(value string) {
	entity.OwnerId = value
	if entity.Has == nil {
		entity.Has = &NamespaceRecordHas{}
	}
	entity.Has.OwnerId = true
}
func (entity *NamespaceRecord) GetName() string {
	return entity.Name
}
func (entity *NamespaceRecord) SetName(value string) {
	entity.Name = value
	if entity.Has == nil {
		entity.Has = &NamespaceRecordHas{}
	}
	entity.Has.Name = true
}
func (entity *NamespaceRecord) GetTitle() string {
	return entity.Title
}
func (entity *NamespaceRecord) SetTitle(value string) {
	entity.Title = value
	if entity.Has == nil {
		entity.Has = &NamespaceRecordHas{}
	}
	entity.Has.Title = true
}
func (entity *NamespaceRecord) GetDescription() *string {
	return entity.Description
}
func (entity *NamespaceRecord) SetDescription(value *string) {
	entity.Description = value
	if entity.Has == nil {
		entity.Has = &NamespaceRecordHas{}
	}
	entity.Has.Description = true
}
func (entity *NamespaceRecord) GetStatus() string {
	return entity.Status
}
func (entity *NamespaceRecord) SetStatus(value string) {
	entity.Status = value
	if entity.Has == nil {
		entity.Has = &NamespaceRecordHas{}
	}
	entity.Has.Status = true
}
func (entity *NamespaceRecord) GetEtag() *int64 {
	return entity.Etag
}
func (entity *NamespaceRecord) SetEtag(value *int64) {
	entity.Etag = value
	if entity.Has == nil {
		entity.Has = &NamespaceRecordHas{}
	}
	entity.Has.Etag = true
}
func (entity *NamespaceRecord) GetCreatedAt() *time.Time {
	return entity.CreatedAt
}
func (entity *NamespaceRecord) SetCreatedAt(value *time.Time) {
	entity.CreatedAt = value
	if entity.Has == nil {
		entity.Has = &NamespaceRecordHas{}
	}
	entity.Has.CreatedAt = true
}
func (entity *NamespaceRecord) GetUpdatedAt() *time.Time {
	return entity.UpdatedAt
}
func (entity *NamespaceRecord) SetUpdatedAt(value *time.Time) {
	entity.UpdatedAt = value
	if entity.Has == nil {
		entity.Has = &NamespaceRecordHas{}
	}
	entity.Has.UpdatedAt = true
}
func (entity *NamespaceRecord) GetDeletedAt() *time.Time {
	return entity.DeletedAt
}
func (entity *NamespaceRecord) SetDeletedAt(value *time.Time) {
	entity.DeletedAt = value
	if entity.Has == nil {
		entity.Has = &NamespaceRecordHas{}
	}
	entity.Has.DeletedAt = true
}
