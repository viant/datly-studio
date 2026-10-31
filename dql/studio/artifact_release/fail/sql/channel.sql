SELECT c.scope_id,c.name,c.revision,c.pending_operation_id,c.updated_at,c.updated_by FROM artifact_release_channels c
${predicate.Builder().CombineAnd($predicate.FilterGroup(0,"AND")).Build("WHERE")}
