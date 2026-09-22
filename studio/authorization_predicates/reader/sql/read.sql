SELECT predicates.* FROM  (SELECT p.name, p.title, p.description, p.package_path, p.type_name, p.sql_scope_json,
       p.owner_id, p.status, p.etag, p.created_at, p.updated_at
FROM authorization_predicates p
WHERE p.deleted_at IS NULL
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(0, "OR"),
    $predicate.FilterGroup(1, "AND"),
    $predicate.FilterGroup(2, "AND"),
    $predicate.FilterGroup(3, "AND")
).Build("AND")}
)  predicates WHERE 1 = 1