SELECT namespace."namespace_id", namespace."visibility", namespace."allowed_roles_json", namespace."mcp_enabled", namespace."mcp_port", namespace."owner_id", namespace."name", namespace."title", namespace."description", namespace."status", namespace."etag", namespace."created_at", namespace."updated_at", namespace."deleted_at" FROM  (SELECT n.namespace_id, n.visibility, n.allowed_roles_json, n.mcp_enabled, n.mcp_port, n.owner_id, n.name, n.title, n.description, n.status, n.etag,
       n.created_at, n.updated_at, n.deleted_at
FROM namespaces n
WHERE n.deleted_at IS NULL
)  namespace WHERE 1 = 1