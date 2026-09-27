SELECT r.id
FROM components r
WHERE r.deleted_at IS NULL
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(3, "AND")
).Build("AND")}
