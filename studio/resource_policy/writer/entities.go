package writer

import (
	json "encoding/json"
	time "time"
)

func (entity *ResourcePolicyHead) GetTenantId() string {
	return entity.TenantId
}
func (entity *ResourcePolicyHead) SetTenantId(value string) {
	entity.TenantId = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyHeadHas{}
	}
	entity.Has.TenantId = true
}
func (entity *ResourcePolicyHead) GetResourceKind() string {
	return entity.ResourceKind
}
func (entity *ResourcePolicyHead) SetResourceKind(value string) {
	entity.ResourceKind = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyHeadHas{}
	}
	entity.Has.ResourceKind = true
}
func (entity *ResourcePolicyHead) GetResourceId() string {
	return entity.ResourceId
}
func (entity *ResourcePolicyHead) SetResourceId(value string) {
	entity.ResourceId = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyHeadHas{}
	}
	entity.Has.ResourceId = true
}
func (entity *ResourcePolicyHead) GetResourceVersion() string {
	return entity.ResourceVersion
}
func (entity *ResourcePolicyHead) SetResourceVersion(value string) {
	entity.ResourceVersion = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyHeadHas{}
	}
	entity.Has.ResourceVersion = true
}
func (entity *ResourcePolicyHead) GetCreatedAt() *time.Time {
	return entity.CreatedAt
}
func (entity *ResourcePolicyHead) SetCreatedAt(value *time.Time) {
	entity.CreatedAt = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyHeadHas{}
	}
	entity.Has.CreatedAt = true
}
func (entity *ResourcePolicyHead) GetCreatedBy() *string {
	return entity.CreatedBy
}
func (entity *ResourcePolicyHead) SetCreatedBy(value *string) {
	entity.CreatedBy = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyHeadHas{}
	}
	entity.Has.CreatedBy = true
}
func (entity *ResourcePolicyHead) GetUpdatedAt() *time.Time {
	return entity.UpdatedAt
}
func (entity *ResourcePolicyHead) SetUpdatedAt(value *time.Time) {
	entity.UpdatedAt = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyHeadHas{}
	}
	entity.Has.UpdatedAt = true
}
func (entity *ResourcePolicyHead) GetUpdatedBy() *string {
	return entity.UpdatedBy
}
func (entity *ResourcePolicyHead) SetUpdatedBy(value *string) {
	entity.UpdatedBy = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyHeadHas{}
	}
	entity.Has.UpdatedBy = true
}
func (entity *ResourcePolicyHead) GetRevision() *int {
	return entity.Revision
}
func (entity *ResourcePolicyHead) SetRevision(value *int) {
	entity.Revision = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyHeadHas{}
	}
	entity.Has.Revision = true
}
func (entity *ResourcePolicyHead) GetHistory() []*ResourcePolicyRevision {
	return entity.History
}
func (entity *ResourcePolicyHead) SetHistory(value []*ResourcePolicyRevision) {
	entity.History = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyHeadHas{}
	}
	entity.Has.History = true
}
func (entity *ResourcePolicyRevision) GetTenantId() string {
	return entity.TenantId
}
func (entity *ResourcePolicyRevision) SetTenantId(value string) {
	entity.TenantId = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyRevisionHas{}
	}
	entity.Has.TenantId = true
}
func (entity *ResourcePolicyRevision) GetResourceKind() string {
	return entity.ResourceKind
}
func (entity *ResourcePolicyRevision) SetResourceKind(value string) {
	entity.ResourceKind = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyRevisionHas{}
	}
	entity.Has.ResourceKind = true
}
func (entity *ResourcePolicyRevision) GetResourceId() string {
	return entity.ResourceId
}
func (entity *ResourcePolicyRevision) SetResourceId(value string) {
	entity.ResourceId = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyRevisionHas{}
	}
	entity.Has.ResourceId = true
}
func (entity *ResourcePolicyRevision) GetResourceVersion() string {
	return entity.ResourceVersion
}
func (entity *ResourcePolicyRevision) SetResourceVersion(value string) {
	entity.ResourceVersion = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyRevisionHas{}
	}
	entity.Has.ResourceVersion = true
}
func (entity *ResourcePolicyRevision) GetActorId() string {
	return entity.ActorId
}
func (entity *ResourcePolicyRevision) SetActorId(value string) {
	entity.ActorId = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyRevisionHas{}
	}
	entity.Has.ActorId = true
}
func (entity *ResourcePolicyRevision) GetCreatedAt() *time.Time {
	return entity.CreatedAt
}
func (entity *ResourcePolicyRevision) SetCreatedAt(value *time.Time) {
	entity.CreatedAt = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyRevisionHas{}
	}
	entity.Has.CreatedAt = true
}
func (entity *ResourcePolicyRevision) GetCreatedBy() *string {
	return entity.CreatedBy
}
func (entity *ResourcePolicyRevision) SetCreatedBy(value *string) {
	entity.CreatedBy = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyRevisionHas{}
	}
	entity.Has.CreatedBy = true
}
func (entity *ResourcePolicyRevision) GetUpdatedAt() *time.Time {
	return entity.UpdatedAt
}
func (entity *ResourcePolicyRevision) SetUpdatedAt(value *time.Time) {
	entity.UpdatedAt = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyRevisionHas{}
	}
	entity.Has.UpdatedAt = true
}
func (entity *ResourcePolicyRevision) GetUpdatedBy() *string {
	return entity.UpdatedBy
}
func (entity *ResourcePolicyRevision) SetUpdatedBy(value *string) {
	entity.UpdatedBy = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyRevisionHas{}
	}
	entity.Has.UpdatedBy = true
}
func (entity *ResourcePolicyRevision) GetPoliciesJson() json.RawMessage {
	return entity.PoliciesJson
}
func (entity *ResourcePolicyRevision) SetPoliciesJson(value json.RawMessage) {
	entity.PoliciesJson = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyRevisionHas{}
	}
	entity.Has.PoliciesJson = true
}
func (entity *ResourcePolicyRevision) GetRevision() *int {
	return entity.Revision
}
func (entity *ResourcePolicyRevision) SetRevision(value *int) {
	entity.Revision = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyRevisionHas{}
	}
	entity.Has.Revision = true
}
func (entity *ResourcePolicyRevision) GetOccurredAt() *time.Time {
	return entity.OccurredAt
}
func (entity *ResourcePolicyRevision) SetOccurredAt(value *time.Time) {
	entity.OccurredAt = value
	if entity.Has == nil {
		entity.Has = &ResourcePolicyRevisionHas{}
	}
	entity.Has.OccurredAt = true
}
func (input *Input) ProjectCurrentHistoryParentKeys(previous []*CurrentPolicyView) ([]struct {
	TenantId        string "sqlx:\"tenant_id\""
	ResourceKind    string "sqlx:\"resource_kind\""
	ResourceId      string "sqlx:\"resource_id\""
	ResourceVersion string "sqlx:\"resource_version\""
}, error) {
	result := []struct {
		TenantId        string "sqlx:\"tenant_id\""
		ResourceKind    string "sqlx:\"resource_kind\""
		ResourceId      string "sqlx:\"resource_id\""
		ResourceVersion string "sqlx:\"resource_version\""
	}{}
	for _, row := range previous {
		if row == nil {
			continue
		}
		result = append(result, struct {
			TenantId        string "sqlx:\"tenant_id\""
			ResourceKind    string "sqlx:\"resource_kind\""
			ResourceId      string "sqlx:\"resource_id\""
			ResourceVersion string "sqlx:\"resource_version\""
		}{TenantId: row.TenantId, ResourceKind: row.ResourceKind, ResourceId: row.ResourceId, ResourceVersion: row.ResourceVersion})
	}
	return result, nil
}
