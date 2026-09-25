SELECT namespace."owner_id", namespace."name" FROM  (SELECT n.owner_id, n.name
FROM namespaces n
WHERE n.deleted_at IS NULL
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(3, "AND")
).Build("AND")}
)  namespace WHERE 1 = 1