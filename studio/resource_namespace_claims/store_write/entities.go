package store_write

import (
	time "time"
)

func (entity *StoredClaim) GetNamespace() string {
	return entity.Namespace
}
func (entity *StoredClaim) SetNamespace(value string) {
	entity.Namespace = value
	if entity.Has == nil {
		entity.Has = &StoredClaimHas{}
	}
	entity.Has.Namespace = true
}
func (entity *StoredClaim) GetReportId() string {
	return entity.ReportId
}
func (entity *StoredClaim) SetReportId(value string) {
	entity.ReportId = value
	if entity.Has == nil {
		entity.Has = &StoredClaimHas{}
	}
	entity.Has.ReportId = true
}
func (entity *StoredClaim) GetCreatedAt() time.Time {
	return entity.CreatedAt
}
func (entity *StoredClaim) SetCreatedAt(value time.Time) {
	entity.CreatedAt = value
	if entity.Has == nil {
		entity.Has = &StoredClaimHas{}
	}
	entity.Has.CreatedAt = true
}
func (entity *StoredClaim) GetCreatedBy() string {
	return entity.CreatedBy
}
func (entity *StoredClaim) SetCreatedBy(value string) {
	entity.CreatedBy = value
	if entity.Has == nil {
		entity.Has = &StoredClaimHas{}
	}
	entity.Has.CreatedBy = true
}
func (entity *StoredClaim) GetUpdatedAt() time.Time {
	return entity.UpdatedAt
}
func (entity *StoredClaim) SetUpdatedAt(value time.Time) {
	entity.UpdatedAt = value
	if entity.Has == nil {
		entity.Has = &StoredClaimHas{}
	}
	entity.Has.UpdatedAt = true
}
func (entity *StoredClaim) GetUpdatedBy() string {
	return entity.UpdatedBy
}
func (entity *StoredClaim) SetUpdatedBy(value string) {
	entity.UpdatedBy = value
	if entity.Has == nil {
		entity.Has = &StoredClaimHas{}
	}
	entity.Has.UpdatedBy = true
}
func (entity *StoredClaim) GetShouldDelete() bool {
	return entity.ShouldDelete
}
func (entity *StoredClaim) SetShouldDelete(value bool) {
	entity.ShouldDelete = value
	if entity.Has == nil {
		entity.Has = &StoredClaimHas{}
	}
	entity.Has.ShouldDelete = true
}
