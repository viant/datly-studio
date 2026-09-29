SELECT namespace."namespace_id", namespace."visibility", namespace."allowed_roles_json", namespace."mcp_enabled", namespace."mcp_port", namespace."owner_id", namespace."name", namespace."title", namespace."description", namespace."status", namespace."etag", namespace."created_at", namespace."updated_at" FROM  (SELECT n.namespace_id, n.visibility, n.allowed_roles_json, n.mcp_enabled, n.mcp_port, n.owner_id, n.name, n.title, n.description, n.status, n.etag,
       n.created_at, n.updated_at
FROM namespaces n
WHERE n.deleted_at IS NULL
 AND ($NamespaceId = '' OR n.namespace_id = $NamespaceId)
  AND ($OwnerId = '' OR n.owner_id = $OwnerId)
  AND ($Name = '' OR n.name = $Name)
  AND ($Query = '' OR LOWER(n.name) LIKE $Query OR LOWER(n.title) LIKE $Query OR LOWER(n.description) LIKE $Query)
  AND ($Status = '' OR n.status = $Status)
${predicate.Builder().CombineAnd($predicate.FilterGroup(3, "AND")).Build("AND")}
ORDER BY n.updated_at DESC, n.name ASC, n.owner_id ASC, n.namespace_id ASC
LIMIT $PageLimit OFFSET $PageOffset
)  namespace WHERE 1 = 1