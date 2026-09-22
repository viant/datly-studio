package writer

import (
	time "time"
)

func (entity *AuthorizationPredicateMutationRecord) GetName() string {
	return entity.Name
}
func (entity *AuthorizationPredicateMutationRecord) SetName(value string) {
	entity.Name = value
	if entity.Has == nil {
		entity.Has = &AuthorizationPredicateMutationRecordHas{}
	}
	entity.Has.Name = true
}
func (entity *AuthorizationPredicateMutationRecord) GetTitle() string {
	return entity.Title
}
func (entity *AuthorizationPredicateMutationRecord) SetTitle(value string) {
	entity.Title = value
	if entity.Has == nil {
		entity.Has = &AuthorizationPredicateMutationRecordHas{}
	}
	entity.Has.Title = true
}
func (entity *AuthorizationPredicateMutationRecord) GetPackagePath() string {
	return entity.PackagePath
}
func (entity *AuthorizationPredicateMutationRecord) SetPackagePath(value string) {
	entity.PackagePath = value
	if entity.Has == nil {
		entity.Has = &AuthorizationPredicateMutationRecordHas{}
	}
	entity.Has.PackagePath = true
}
func (entity *AuthorizationPredicateMutationRecord) GetTypeName() string {
	return entity.TypeName
}
func (entity *AuthorizationPredicateMutationRecord) SetTypeName(value string) {
	entity.TypeName = value
	if entity.Has == nil {
		entity.Has = &AuthorizationPredicateMutationRecordHas{}
	}
	entity.Has.TypeName = true
}
func (entity *AuthorizationPredicateMutationRecord) GetSqlScopeJson() *string {
	return entity.SqlScopeJson
}
func (entity *AuthorizationPredicateMutationRecord) SetSqlScopeJson(value *string) {
	entity.SqlScopeJson = value
	if entity.Has == nil {
		entity.Has = &AuthorizationPredicateMutationRecordHas{}
	}
	entity.Has.SqlScopeJson = true
}
func (entity *AuthorizationPredicateMutationRecord) GetOwnerId() string {
	return entity.OwnerId
}
func (entity *AuthorizationPredicateMutationRecord) SetOwnerId(value string) {
	entity.OwnerId = value
	if entity.Has == nil {
		entity.Has = &AuthorizationPredicateMutationRecordHas{}
	}
	entity.Has.OwnerId = true
}
func (entity *AuthorizationPredicateMutationRecord) GetStatus() string {
	return entity.Status
}
func (entity *AuthorizationPredicateMutationRecord) SetStatus(value string) {
	entity.Status = value
	if entity.Has == nil {
		entity.Has = &AuthorizationPredicateMutationRecordHas{}
	}
	entity.Has.Status = true
}
func (entity *AuthorizationPredicateMutationRecord) GetShouldDelete() bool {
	return entity.ShouldDelete
}
func (entity *AuthorizationPredicateMutationRecord) GetEtag() *int {
	return entity.Etag
}
func (entity *AuthorizationPredicateMutationRecord) SetEtag(value *int) {
	entity.Etag = value
	if entity.Has == nil {
		entity.Has = &AuthorizationPredicateMutationRecordHas{}
	}
	entity.Has.Etag = true
}
func (entity *AuthorizationPredicateMutationRecord) GetDescription() *string {
	return entity.Description
}
func (entity *AuthorizationPredicateMutationRecord) SetDescription(value *string) {
	entity.Description = value
	if entity.Has == nil {
		entity.Has = &AuthorizationPredicateMutationRecordHas{}
	}
	entity.Has.Description = true
}
func (entity *AuthorizationPredicateMutationRecord) GetCreatedAt() *time.Time {
	return entity.CreatedAt
}
func (entity *AuthorizationPredicateMutationRecord) SetCreatedAt(value *time.Time) {
	entity.CreatedAt = value
	if entity.Has == nil {
		entity.Has = &AuthorizationPredicateMutationRecordHas{}
	}
	entity.Has.CreatedAt = true
}
func (entity *AuthorizationPredicateMutationRecord) GetUpdatedAt() *time.Time {
	return entity.UpdatedAt
}
func (entity *AuthorizationPredicateMutationRecord) SetUpdatedAt(value *time.Time) {
	entity.UpdatedAt = value
	if entity.Has == nil {
		entity.Has = &AuthorizationPredicateMutationRecordHas{}
	}
	entity.Has.UpdatedAt = true
}
func (entity *AuthorizationPredicateMutationRecord) GetDeletedAt() *time.Time {
	return entity.DeletedAt
}
func (entity *AuthorizationPredicateMutationRecord) SetDeletedAt(value *time.Time) {
	entity.DeletedAt = value
	if entity.Has == nil {
		entity.Has = &AuthorizationPredicateMutationRecordHas{}
	}
	entity.Has.DeletedAt = true
}
