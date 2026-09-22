package writer

import (
	json "encoding/json"
	time "time"
)

func (entity *Connector) GetOptionsJson() json.RawMessage {
	return entity.OptionsJson
}
func (entity *Connector) SetOptionsJson(value json.RawMessage) {
	entity.OptionsJson = value
	if entity.Has == nil {
		entity.Has = &ConnectorHas{}
	}
	entity.Has.OptionsJson = true
}
func (entity *Connector) GetName() *string {
	return entity.Name
}
func (entity *Connector) SetName(value *string) {
	entity.Name = value
	if entity.Has == nil {
		entity.Has = &ConnectorHas{}
	}
	entity.Has.Name = true
}
func (entity *Connector) GetDriver() *string {
	return entity.Driver
}
func (entity *Connector) SetDriver(value *string) {
	entity.Driver = value
	if entity.Has == nil {
		entity.Has = &ConnectorHas{}
	}
	entity.Has.Driver = true
}
func (entity *Connector) GetDsnTemplate() *string {
	return entity.DsnTemplate
}
func (entity *Connector) SetDsnTemplate(value *string) {
	entity.DsnTemplate = value
	if entity.Has == nil {
		entity.Has = &ConnectorHas{}
	}
	entity.Has.DsnTemplate = true
}
func (entity *Connector) GetSecretRef() *string {
	return entity.SecretRef
}
func (entity *Connector) SetSecretRef(value *string) {
	entity.SecretRef = value
	if entity.Has == nil {
		entity.Has = &ConnectorHas{}
	}
	entity.Has.SecretRef = true
}
func (entity *Connector) GetShouldDelete() bool {
	return entity.ShouldDelete
}
func (entity *Connector) GetEtag() *int {
	return entity.Etag
}
func (entity *Connector) SetEtag(value *int) {
	entity.Etag = value
	if entity.Has == nil {
		entity.Has = &ConnectorHas{}
	}
	entity.Has.Etag = true
}
func (entity *Connector) GetDescription() *string {
	return entity.Description
}
func (entity *Connector) SetDescription(value *string) {
	entity.Description = value
	if entity.Has == nil {
		entity.Has = &ConnectorHas{}
	}
	entity.Has.Description = true
}
func (entity *Connector) GetOwnerId() *string {
	return entity.OwnerId
}
func (entity *Connector) SetOwnerId(value *string) {
	entity.OwnerId = value
	if entity.Has == nil {
		entity.Has = &ConnectorHas{}
	}
	entity.Has.OwnerId = true
}
func (entity *Connector) GetStatus() *string {
	return entity.Status
}
func (entity *Connector) SetStatus(value *string) {
	entity.Status = value
	if entity.Has == nil {
		entity.Has = &ConnectorHas{}
	}
	entity.Has.Status = true
}
func (entity *Connector) GetLastTestStatus() *string {
	return entity.LastTestStatus
}
func (entity *Connector) SetLastTestStatus(value *string) {
	entity.LastTestStatus = value
	if entity.Has == nil {
		entity.Has = &ConnectorHas{}
	}
	entity.Has.LastTestStatus = true
}
func (entity *Connector) GetLastTestErrorCode() *string {
	return entity.LastTestErrorCode
}
func (entity *Connector) SetLastTestErrorCode(value *string) {
	entity.LastTestErrorCode = value
	if entity.Has == nil {
		entity.Has = &ConnectorHas{}
	}
	entity.Has.LastTestErrorCode = true
}
func (entity *Connector) GetLastTestedAt() *time.Time {
	return entity.LastTestedAt
}
func (entity *Connector) SetLastTestedAt(value *time.Time) {
	entity.LastTestedAt = value
	if entity.Has == nil {
		entity.Has = &ConnectorHas{}
	}
	entity.Has.LastTestedAt = true
}
func (entity *Connector) GetCreatedAt() *time.Time {
	return entity.CreatedAt
}
func (entity *Connector) SetCreatedAt(value *time.Time) {
	entity.CreatedAt = value
	if entity.Has == nil {
		entity.Has = &ConnectorHas{}
	}
	entity.Has.CreatedAt = true
}
func (entity *Connector) GetUpdatedAt() *time.Time {
	return entity.UpdatedAt
}
func (entity *Connector) SetUpdatedAt(value *time.Time) {
	entity.UpdatedAt = value
	if entity.Has == nil {
		entity.Has = &ConnectorHas{}
	}
	entity.Has.UpdatedAt = true
}
func (entity *Connector) GetDeletedAt() *time.Time {
	return entity.DeletedAt
}
func (entity *Connector) SetDeletedAt(value *time.Time) {
	entity.DeletedAt = value
	if entity.Has == nil {
		entity.Has = &ConnectorHas{}
	}
	entity.Has.DeletedAt = true
}
