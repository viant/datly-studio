SELECT generation."generation_no", generation."status" FROM  (SELECT g.generation_no, g.status
FROM runtime_generations g
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(3, "AND")
).Build("WHERE")}
)  generation WHERE 1 = 1