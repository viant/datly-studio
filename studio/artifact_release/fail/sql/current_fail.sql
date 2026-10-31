SELECT r.scope_id, r.name, r.revision, r.pending_operation_id, r.updated_at, r.updated_by FROM (SELECT * FROM  (SELECT c.scope_id,c.name,c.revision,c.pending_operation_id,c.updated_at,c.updated_by FROM artifact_release_channels c
${predicate.Builder().CombineAnd($predicate.FilterGroup(0,"AND")).Build("WHERE")}
)  channel WHERE 1 = 1) r WHERE $criteria.CompositeIn("r", $FailKeys)