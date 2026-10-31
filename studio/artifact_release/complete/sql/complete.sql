SELECT * FROM  (SELECT c.scope_id,c.name,c.revision,c.pending_operation_id,c.updated_at,c.updated_by,c.description,c.policy_refs_json,c.active_bundle_hash,c.active_dependencies_json FROM artifact_release_channels c
${predicate.Builder().CombineAnd($predicate.FilterGroup(0,"AND")).Build("WHERE")}
)  channel WHERE 1 = 1