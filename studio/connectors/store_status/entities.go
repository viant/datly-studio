package store_status

import (
	time "time"
)

func (entity *StoredConnector) GetName() string {
	return entity.Name
}
func (entity *StoredConnector) SetName(value string) {
	entity.Name = value
	if entity.Has == nil {
		entity.Has = &StoredConnectorHas{}
	}
	entity.Has.Name = true
}
func (entity *StoredConnector) GetStatus() string {
	return entity.Status
}
func (entity *StoredConnector) SetStatus(value string) {
	entity.Status = value
	if entity.Has == nil {
		entity.Has = &StoredConnectorHas{}
	}
	entity.Has.Status = true
}
func (entity *StoredConnector) GetEtag() *int64 {
	return entity.Etag
}
func (entity *StoredConnector) SetEtag(value *int64) {
	entity.Etag = value
	if entity.Has == nil {
		entity.Has = &StoredConnectorHas{}
	}
	entity.Has.Etag = true
}
func (entity *StoredConnector) GetUpdatedAt() *time.Time {
	return entity.UpdatedAt
}
func (entity *StoredConnector) SetUpdatedAt(value *time.Time) {
	entity.UpdatedAt = value
	if entity.Has == nil {
		entity.Has = &StoredConnectorHas{}
	}
	entity.Has.UpdatedAt = true
}
func (entity *StoredConnector) GetLastTestStatus() *string {
	return entity.LastTestStatus
}
func (entity *StoredConnector) SetLastTestStatus(value *string) {
	entity.LastTestStatus = value
	if entity.Has == nil {
		entity.Has = &StoredConnectorHas{}
	}
	entity.Has.LastTestStatus = true
}
func (entity *StoredConnector) GetLastTestErrorCode() *string {
	return entity.LastTestErrorCode
}
func (entity *StoredConnector) SetLastTestErrorCode(value *string) {
	entity.LastTestErrorCode = value
	if entity.Has == nil {
		entity.Has = &StoredConnectorHas{}
	}
	entity.Has.LastTestErrorCode = true
}
func (entity *StoredConnector) GetLastTestedAt() *time.Time {
	return entity.LastTestedAt
}
func (entity *StoredConnector) SetLastTestedAt(value *time.Time) {
	entity.LastTestedAt = value
	if entity.Has == nil {
		entity.Has = &StoredConnectorHas{}
	}
	entity.Has.LastTestedAt = true
}
func (entity *StoredConnector) GetDeletedAt() *time.Time {
	return entity.DeletedAt
}
func (entity *StoredConnector) SetDeletedAt(value *time.Time) {
	entity.DeletedAt = value
	if entity.Has == nil {
		entity.Has = &StoredConnectorHas{}
	}
	entity.Has.DeletedAt = true
}
