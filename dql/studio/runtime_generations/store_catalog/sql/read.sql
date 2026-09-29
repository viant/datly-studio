SELECT g.generation_no, g.status, g.source_revision, g.requested_at
FROM runtime_generations g
WHERE g.namespace_id=$NamespaceId
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(1, "AND")
).Build("AND")}
