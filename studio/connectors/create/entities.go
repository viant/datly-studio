package create

import (
	json "encoding/json"
	time "time"
)

func (entity *ConnectorRecord) GetName() string {
	return entity.Name
}
func (entity *ConnectorRecord) SetName(value string) {
	entity.Name = value
	if entity.Has == nil {
		entity.Has = &ConnectorRecordHas{}
	}
	entity.Has.Name = true
}
func (entity *ConnectorRecord) GetDriver() string {
	return entity.Driver
}
func (entity *ConnectorRecord) SetDriver(value string) {
	entity.Driver = value
	if entity.Has == nil {
		entity.Has = &ConnectorRecordHas{}
	}
	entity.Has.Driver = true
}
func (entity *ConnectorRecord) GetDsnTemplate() *string {
	return entity.DsnTemplate
}
func (entity *ConnectorRecord) SetDsnTemplate(value *string) {
	entity.DsnTemplate = value
	if entity.Has == nil {
		entity.Has = &ConnectorRecordHas{}
	}
	entity.Has.DsnTemplate = true
}
func (entity *ConnectorRecord) GetSecretRef() *string {
	return entity.SecretRef
}
func (entity *ConnectorRecord) SetSecretRef(value *string) {
	entity.SecretRef = value
	if entity.Has == nil {
		entity.Has = &ConnectorRecordHas{}
	}
	entity.Has.SecretRef = true
}
func (entity *ConnectorRecord) GetDescription() *string {
	return entity.Description
}
func (entity *ConnectorRecord) SetDescription(value *string) {
	entity.Description = value
	if entity.Has == nil {
		entity.Has = &ConnectorRecordHas{}
	}
	entity.Has.Description = true
}
func (entity *ConnectorRecord) GetOwnerId() string {
	return entity.OwnerId
}
func (entity *ConnectorRecord) SetOwnerId(value string) {
	entity.OwnerId = value
	if entity.Has == nil {
		entity.Has = &ConnectorRecordHas{}
	}
	entity.Has.OwnerId = true
}
func (entity *ConnectorRecord) GetStatus() string {
	return entity.Status
}
func (entity *ConnectorRecord) SetStatus(value string) {
	entity.Status = value
	if entity.Has == nil {
		entity.Has = &ConnectorRecordHas{}
	}
	entity.Has.Status = true
}
func (entity *ConnectorRecord) GetOptionsJson() json.RawMessage {
	return entity.OptionsJson
}
func (entity *ConnectorRecord) SetOptionsJson(value json.RawMessage) {
	entity.OptionsJson = value
	if entity.Has == nil {
		entity.Has = &ConnectorRecordHas{}
	}
	entity.Has.OptionsJson = true
}
func (entity *ConnectorRecord) GetEtag() int64 {
	return entity.Etag
}
func (entity *ConnectorRecord) SetEtag(value int64) {
	entity.Etag = value
	if entity.Has == nil {
		entity.Has = &ConnectorRecordHas{}
	}
	entity.Has.Etag = true
}
func (entity *ConnectorRecord) GetCreatedAt() time.Time {
	return entity.CreatedAt
}
func (entity *ConnectorRecord) SetCreatedAt(value time.Time) {
	entity.CreatedAt = value
	if entity.Has == nil {
		entity.Has = &ConnectorRecordHas{}
	}
	entity.Has.CreatedAt = true
}
func (entity *ConnectorRecord) GetUpdatedAt() time.Time {
	return entity.UpdatedAt
}
func (entity *ConnectorRecord) SetUpdatedAt(value time.Time) {
	entity.UpdatedAt = value
	if entity.Has == nil {
		entity.Has = &ConnectorRecordHas{}
	}
	entity.Has.UpdatedAt = true
}
