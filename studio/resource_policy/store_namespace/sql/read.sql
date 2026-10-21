SELECT namespace.* FROM  (SELECT n.namespace_id, n.owner_id, CAST(n.allowed_roles_json AS CHAR) AS roles_json, n.status
FROM namespaces n
WHERE (($NamespaceId <> '' AND n.namespace_id = $NamespaceId AND $ComponentId = '')
 OR ($NamespaceId = '' AND EXISTS (SELECT 1 FROM components c WHERE c.id = $ComponentId AND c.namespace_id = n.namespace_id)))
)  namespace WHERE 1 = 1