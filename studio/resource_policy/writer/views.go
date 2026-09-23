package writer

import (
	json "encoding/json"
	time "time"
)

// ResourcePolicyHead is generated canonical view metadata for policy.
type ResourcePolicyHead struct {
	TenantId        string                    `sqlx:"tenant_id,primaryKey" validate:"required"`
	ResourceKind    string                    `sqlx:"resource_kind,primaryKey" validate:"required"`
	ResourceId      string                    `sqlx:"resource_id,primaryKey" validate:"required"`
	ResourceVersion string                    `sqlx:"resource_version,primaryKey" validate:"required"`
	Revision        *int                      `writer:"concurrency" sqlx:"revision"`
	History         []*ResourcePolicyRevision `view:"history,type=ResourcePolicyRevision,table=resource_policy_revisions" on:"TenantId:head.tenant_id=TenantId:history.tenant_id,ResourceKind:head.resource_kind=ResourceKind:history.resource_kind,ResourceId:head.resource_id=ResourceId:history.resource_id,ResourceVersion:head.resource_version=ResourceVersion:history.resource_version" json:"history" sql:"uri=studio_resource_policy_writer_policy:sql/history.sql"`
	Has             *ResourcePolicyHeadHas    `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"ResourcePolicyHeadHas"`
}

type ResourcePolicyHeadHas struct {
	TenantId        bool
	ResourceKind    bool
	ResourceId      bool
	ResourceVersion bool
	Revision        bool
	History         bool
}

// ResourcePolicyRevision is generated canonical view metadata for policy.
type ResourcePolicyRevision struct {
	TenantId        string                     `sqlx:"tenant_id,primaryKey"`
	ResourceKind    string                     `sqlx:"resource_kind,primaryKey"`
	ResourceId      string                     `sqlx:"resource_id,primaryKey"`
	ResourceVersion string                     `sqlx:"resource_version,primaryKey"`
	ActorId         string                     `validate:"required" sqlx:"actor_id"`
	PoliciesJson    json.RawMessage            `sqlx:"policies_json,enc=JSON" validate:"required"`
	Revision        *int                       `sqlx:"revision,primaryKey"`
	OccurredAt      *time.Time                 `validate:"required" sqlx:"occurred_at"`
	Has             *ResourcePolicyRevisionHas `setMarker:"true" format:"-" sqlx:"-" diff:"-" json:"-" typeName:"ResourcePolicyRevisionHas"`
}

type ResourcePolicyRevisionHas struct {
	TenantId        bool
	ResourceKind    bool
	ResourceId      bool
	ResourceVersion bool
	ActorId         bool
	PoliciesJson    bool
	Revision        bool
	OccurredAt      bool
}

// CurrentPolicyView is generated canonical view metadata for policy.
type CurrentPolicyView struct {
	TenantId        string `sqlx:"tenant_id,primaryKey" validate:"required"`
	ResourceKind    string `sqlx:"resource_kind,primaryKey" validate:"required"`
	ResourceId      string `sqlx:"resource_id,primaryKey" validate:"required"`
	ResourceVersion string `sqlx:"resource_version,primaryKey" validate:"required"`
	Revision        *int   `sqlx:"revision"`
}

// CurrentHistoryView is generated canonical view metadata for policy.
type CurrentHistoryView struct {
	TenantId        string          `sqlx:"tenant_id,primaryKey"`
	ResourceKind    string          `sqlx:"resource_kind,primaryKey"`
	ResourceId      string          `sqlx:"resource_id,primaryKey"`
	ResourceVersion string          `sqlx:"resource_version,primaryKey"`
	ActorId         string          `validate:"required" sqlx:"actor_id"`
	PoliciesJson    json.RawMessage `sqlx:"policies_json,enc=JSON" validate:"required"`
	Revision        *int            `sqlx:"revision,primaryKey"`
	OccurredAt      *time.Time      `validate:"required" sqlx:"occurred_at"`
}

type PolicyKeysRow struct {
	TenantId        string `sqlx:"tenant_id"`
	ResourceKind    string `sqlx:"resource_kind"`
	ResourceId      string `sqlx:"resource_id"`
	ResourceVersion string `sqlx:"resource_version"`
}
