SELECT connector."name" FROM  (SELECT c.name
FROM connectors c
WHERE c.deleted_at IS NULL
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(3, "AND")
).Build("AND")}
)  connector WHERE 1 = 1