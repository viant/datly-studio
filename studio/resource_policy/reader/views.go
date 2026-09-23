package reader

import (
	json "encoding/json"
	time "time"
)

// ResourcePolicy is generated canonical view metadata for policy.
type ResourcePolicy struct {
	TenantId        string          `sqlx:"tenant_id"`
	ResourceKind    string          `sqlx:"resource_kind"`
	ResourceId      string          `sqlx:"resource_id"`
	ResourceVersion string          `sqlx:"resource_version"`
	ActorId         string          `sqlx:"actor_id"`
	PoliciesJson    json.RawMessage `sqlx:"policies_json,enc=JSON"`
	Revision        *int            `sqlx:"revision"`
	OccurredAt      *time.Time      `sqlx:"occurred_at"`
}
