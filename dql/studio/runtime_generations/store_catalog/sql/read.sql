SELECT g.generation_no, g.status, g.source_revision, g.requested_at
FROM runtime_generations g
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(1, "AND")
).Build("WHERE")}
