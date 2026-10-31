SELECT o.operation_id,o.scope_id,o.channel_name,o.status,o.completed_at,o.updated_at,o.updated_by,o.failure_code FROM artifact_release_operations o
${predicate.Builder().CombineAnd($predicate.FilterGroup(1,"AND")).Build("WHERE")}
