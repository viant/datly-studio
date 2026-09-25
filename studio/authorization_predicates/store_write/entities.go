package store_write

import (
	time "time"
)

func (entity *StoredAuthorizationPredicate) GetName() string {
	return entity.Name
}
func (entity *StoredAuthorizationPredicate) SetName(value string) {
	entity.Name = value
	if entity.Has == nil {
		entity.Has = &StoredAuthorizationPredicateHas{}
	}
	entity.Has.Name = true
}
func (entity *StoredAuthorizationPredicate) GetTitle() string {
	return entity.Title
}
func (entity *StoredAuthorizationPredicate) SetTitle(value string) {
	entity.Title = value
	if entity.Has == nil {
		entity.Has = &StoredAuthorizationPredicateHas{}
	}
	entity.Has.Title = true
}
func (entity *StoredAuthorizationPredicate) GetPackagePath() string {
	return entity.PackagePath
}
func (entity *StoredAuthorizationPredicate) SetPackagePath(value string) {
	entity.PackagePath = value
	if entity.Has == nil {
		entity.Has = &StoredAuthorizationPredicateHas{}
	}
	entity.Has.PackagePath = true
}
func (entity *StoredAuthorizationPredicate) GetTypeName() string {
	return entity.TypeName
}
func (entity *StoredAuthorizationPredicate) SetTypeName(value string) {
	entity.TypeName = value
	if entity.Has == nil {
		entity.Has = &StoredAuthorizationPredicateHas{}
	}
	entity.Has.TypeName = true
}
func (entity *StoredAuthorizationPredicate) GetSqlScopeJson() *string {
	return entity.SqlScopeJson
}
func (entity *StoredAuthorizationPredicate) SetSqlScopeJson(value *string) {
	entity.SqlScopeJson = value
	if entity.Has == nil {
		entity.Has = &StoredAuthorizationPredicateHas{}
	}
	entity.Has.SqlScopeJson = true
}
func (entity *StoredAuthorizationPredicate) GetOwnerId() string {
	return entity.OwnerId
}
func (entity *StoredAuthorizationPredicate) SetOwnerId(value string) {
	entity.OwnerId = value
	if entity.Has == nil {
		entity.Has = &StoredAuthorizationPredicateHas{}
	}
	entity.Has.OwnerId = true
}
func (entity *StoredAuthorizationPredicate) GetStatus() string {
	return entity.Status
}
func (entity *StoredAuthorizationPredicate) SetStatus(value string) {
	entity.Status = value
	if entity.Has == nil {
		entity.Has = &StoredAuthorizationPredicateHas{}
	}
	entity.Has.Status = true
}
func (entity *StoredAuthorizationPredicate) GetEtag() *int {
	return entity.Etag
}
func (entity *StoredAuthorizationPredicate) SetEtag(value *int) {
	entity.Etag = value
	if entity.Has == nil {
		entity.Has = &StoredAuthorizationPredicateHas{}
	}
	entity.Has.Etag = true
}
func (entity *StoredAuthorizationPredicate) GetDescription() *string {
	return entity.Description
}
func (entity *StoredAuthorizationPredicate) SetDescription(value *string) {
	entity.Description = value
	if entity.Has == nil {
		entity.Has = &StoredAuthorizationPredicateHas{}
	}
	entity.Has.Description = true
}
func (entity *StoredAuthorizationPredicate) GetCreatedAt() *time.Time {
	return entity.CreatedAt
}
func (entity *StoredAuthorizationPredicate) SetCreatedAt(value *time.Time) {
	entity.CreatedAt = value
	if entity.Has == nil {
		entity.Has = &StoredAuthorizationPredicateHas{}
	}
	entity.Has.CreatedAt = true
}
func (entity *StoredAuthorizationPredicate) GetUpdatedAt() *time.Time {
	return entity.UpdatedAt
}
func (entity *StoredAuthorizationPredicate) SetUpdatedAt(value *time.Time) {
	entity.UpdatedAt = value
	if entity.Has == nil {
		entity.Has = &StoredAuthorizationPredicateHas{}
	}
	entity.Has.UpdatedAt = true
}
func (entity *StoredAuthorizationPredicate) GetDeletedAt() *time.Time {
	return entity.DeletedAt
}
func (entity *StoredAuthorizationPredicate) SetDeletedAt(value *time.Time) {
	entity.DeletedAt = value
	if entity.Has == nil {
		entity.Has = &StoredAuthorizationPredicateHas{}
	}
	entity.Has.DeletedAt = true
}
