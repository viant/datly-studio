SELECT h.tenant_id, h.resource_kind, h.resource_id, h.resource_version, h.revision,
       r.policies_json, r.actor_id, r.occurred_at
FROM resource_policy_heads h
JOIN resource_policy_revisions r
  ON r.tenant_id = h.tenant_id
 AND r.resource_kind = h.resource_kind
 AND r.resource_id = h.resource_id
 AND r.resource_version = h.resource_version
 AND r.revision = h.revision
${predicate.Builder().CombineAnd($predicate.FilterGroup(0, "AND")).Build("WHERE")}
