SELECT namespaces.* FROM  (
    SELECT n.owner_id, n.name, n.title, n.description, n.status,
       n.etag, n.created_at, n.updated_at
FROM namespaces n
WHERE n.deleted_at IS NULL
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(0, "OR"),
    $predicate.FilterGroup(1, "AND")
).Build("AND")}

)  namespaces WHERE 1 = 1 ${predicate.Builder().CombineAnd($predicate.FilterGroup(3, "AND")).Build("AND")} ORDER BY namespaces.updated_at DESC, namespaces.name ASC