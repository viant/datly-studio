SELECT generation."generation_no", generation."status", generation."source_revision", generation."requested_at" FROM  (SELECT g.generation_no, g.status, g.source_revision, g.requested_at
FROM runtime_generations g
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(1, "AND")
).Build("WHERE")}
)  generation WHERE 1 = 1