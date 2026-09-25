SELECT c.name
FROM connectors c
WHERE c.deleted_at IS NULL
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(3, "AND")
).Build("AND")}
