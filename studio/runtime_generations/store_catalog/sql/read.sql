SELECT generation."generation_no", generation."status", generation."source_revision", generation."requested_at" FROM  (SELECT g.generation_no, g.status, g.source_revision, g.requested_at
FROM runtime_generations g
WHERE g.namespace_id=$NamespaceId
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(1, "AND")
).Build("AND")}
)  generation WHERE 1 = 1