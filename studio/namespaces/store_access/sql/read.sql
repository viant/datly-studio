SELECT namespace."namespace_id", namespace."visibility", namespace."allowed_roles_json", namespace."mcp_enabled", namespace."mcp_port", namespace."owner_id", namespace."name" FROM  (SELECT n.namespace_id, n.visibility, n.allowed_roles_json, n.mcp_enabled, n.mcp_port, n.owner_id, n.name
FROM namespaces n
WHERE n.deleted_at IS NULL
${predicate.Builder().CombineAnd(
    $predicate.FilterGroup(3, "AND")
).Build("AND")}
)  namespace WHERE 1 = 1