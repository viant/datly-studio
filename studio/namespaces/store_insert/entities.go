package store_insert

import (
	time "time"
)

func (entity *StoredNamespace) GetOwnerId() string {
	return entity.OwnerId
}
func (entity *StoredNamespace) SetOwnerId(value string) {
	entity.OwnerId = value
	if entity.Has == nil {
		entity.Has = &StoredNamespaceHas{}
	}
	entity.Has.OwnerId = true
}
func (entity *StoredNamespace) GetName() string {
	return entity.Name
}
func (entity *StoredNamespace) SetName(value string) {
	entity.Name = value
	if entity.Has == nil {
		entity.Has = &StoredNamespaceHas{}
	}
	entity.Has.Name = true
}
func (entity *StoredNamespace) GetTitle() string {
	return entity.Title
}
func (entity *StoredNamespace) SetTitle(value string) {
	entity.Title = value
	if entity.Has == nil {
		entity.Has = &StoredNamespaceHas{}
	}
	entity.Has.Title = true
}
func (entity *StoredNamespace) GetDescription() *string {
	return entity.Description
}
func (entity *StoredNamespace) SetDescription(value *string) {
	entity.Description = value
	if entity.Has == nil {
		entity.Has = &StoredNamespaceHas{}
	}
	entity.Has.Description = true
}
func (entity *StoredNamespace) GetStatus() string {
	return entity.Status
}
func (entity *StoredNamespace) SetStatus(value string) {
	entity.Status = value
	if entity.Has == nil {
		entity.Has = &StoredNamespaceHas{}
	}
	entity.Has.Status = true
}
func (entity *StoredNamespace) GetEtag() *int64 {
	return entity.Etag
}
func (entity *StoredNamespace) SetEtag(value *int64) {
	entity.Etag = value
	if entity.Has == nil {
		entity.Has = &StoredNamespaceHas{}
	}
	entity.Has.Etag = true
}
func (entity *StoredNamespace) GetCreatedAt() *time.Time {
	return entity.CreatedAt
}
func (entity *StoredNamespace) SetCreatedAt(value *time.Time) {
	entity.CreatedAt = value
	if entity.Has == nil {
		entity.Has = &StoredNamespaceHas{}
	}
	entity.Has.CreatedAt = true
}
func (entity *StoredNamespace) GetUpdatedAt() *time.Time {
	return entity.UpdatedAt
}
func (entity *StoredNamespace) SetUpdatedAt(value *time.Time) {
	entity.UpdatedAt = value
	if entity.Has == nil {
		entity.Has = &StoredNamespaceHas{}
	}
	entity.Has.UpdatedAt = true
}
func (entity *StoredNamespace) GetDeletedAt() *time.Time {
	return entity.DeletedAt
}
func (entity *StoredNamespace) SetDeletedAt(value *time.Time) {
	entity.DeletedAt = value
	if entity.Has == nil {
		entity.Has = &StoredNamespaceHas{}
	}
	entity.Has.DeletedAt = true
}
