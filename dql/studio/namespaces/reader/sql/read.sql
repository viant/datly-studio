SELECT n.owner_id, n.name, n.title, n.description, n.status,
       n.etag, n.created_at, n.updated_at
FROM namespaces n
WHERE n.deleted_at IS NULL
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(0, "OR"),
    $predicate.FilterGroup(1, "AND")
).Build("AND")}
