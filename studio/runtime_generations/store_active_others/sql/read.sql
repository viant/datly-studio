SELECT generation."generation_no", generation."status" FROM  (SELECT g.generation_no, g.status
FROM runtime_generations g
WHERE g.namespace_id=$NamespaceId
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(3, "AND")
).Build("AND")}
)  generation WHERE 1 = 1