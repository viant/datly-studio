package store_insert

import (
	json "encoding/json"
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
func (entity *StoredConnector) GetDriver() string {
	return entity.Driver
}
func (entity *StoredConnector) SetDriver(value string) {
	entity.Driver = value
	if entity.Has == nil {
		entity.Has = &StoredConnectorHas{}
	}
	entity.Has.Driver = true
}
func (entity *StoredConnector) GetDsnTemplate() *string {
	return entity.DsnTemplate
}
func (entity *StoredConnector) SetDsnTemplate(value *string) {
	entity.DsnTemplate = value
	if entity.Has == nil {
		entity.Has = &StoredConnectorHas{}
	}
	entity.Has.DsnTemplate = true
}
func (entity *StoredConnector) GetSecretRef() *string {
	return entity.SecretRef
}
func (entity *StoredConnector) SetSecretRef(value *string) {
	entity.SecretRef = value
	if entity.Has == nil {
		entity.Has = &StoredConnectorHas{}
	}
	entity.Has.SecretRef = true
}
func (entity *StoredConnector) GetDescription() *string {
	return entity.Description
}
func (entity *StoredConnector) SetDescription(value *string) {
	entity.Description = value
	if entity.Has == nil {
		entity.Has = &StoredConnectorHas{}
	}
	entity.Has.Description = true
}
func (entity *StoredConnector) GetOwnerId() string {
	return entity.OwnerId
}
func (entity *StoredConnector) SetOwnerId(value string) {
	entity.OwnerId = value
	if entity.Has == nil {
		entity.Has = &StoredConnectorHas{}
	}
	entity.Has.OwnerId = true
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
func (entity *StoredConnector) GetOptionsJson() json.RawMessage {
	return entity.OptionsJson
}
func (entity *StoredConnector) SetOptionsJson(value json.RawMessage) {
	entity.OptionsJson = value
	if entity.Has == nil {
		entity.Has = &StoredConnectorHas{}
	}
	entity.Has.OptionsJson = true
}
func (entity *StoredConnector) GetEtag() int64 {
	return entity.Etag
}
func (entity *StoredConnector) SetEtag(value int64) {
	entity.Etag = value
	if entity.Has == nil {
		entity.Has = &StoredConnectorHas{}
	}
	entity.Has.Etag = true
}
func (entity *StoredConnector) GetCreatedAt() time.Time {
	return entity.CreatedAt
}
func (entity *StoredConnector) SetCreatedAt(value time.Time) {
	entity.CreatedAt = value
	if entity.Has == nil {
		entity.Has = &StoredConnectorHas{}
	}
	entity.Has.CreatedAt = true
}
func (entity *StoredConnector) GetUpdatedAt() time.Time {
	return entity.UpdatedAt
}
func (entity *StoredConnector) SetUpdatedAt(value time.Time) {
	entity.UpdatedAt = value
	if entity.Has == nil {
		entity.Has = &StoredConnectorHas{}
	}
	entity.Has.UpdatedAt = true
}
