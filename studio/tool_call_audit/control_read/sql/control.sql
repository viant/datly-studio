SELECT * FROM  (SELECT namespace_id,owner_id,allowed_roles_json,visibility,status,audit_enabled
FROM namespaces WHERE namespace_id=$NamespaceId AND deleted_at IS NULL
)  scope WHERE 1 = 1