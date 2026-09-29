SELECT g.generation_no, g.status
FROM runtime_generations g
WHERE g.namespace_id=$NamespaceId
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(3, "AND")
).Build("AND")}
