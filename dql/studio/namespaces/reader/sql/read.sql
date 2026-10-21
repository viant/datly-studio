SELECT '' AS allowed_roles, 0 AS can_manage, '' AS manager_roles, n.namespace_id, n.visibility, CAST(n.allowed_roles_json AS CHAR) AS allowed_roles_json, n.mcp_enabled, n.mcp_port, n.owner_id, n.name, n.title, n.description, n.status,
       n.etag, n.created_at, n.updated_at
FROM namespaces n
WHERE n.deleted_at IS NULL
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(0, "OR"),
    $predicate.FilterGroup(1, "AND")
).Build("AND")}
