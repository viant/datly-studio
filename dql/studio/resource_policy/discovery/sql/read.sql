SELECT s.resource_kind, s.resource_id, COALESCE(p.resource_version, '') AS resource_version,
       COALESCE(p.revision, 0) AS revision, v.policies_json
FROM (
  SELECT 'namespace' AS resource_kind, n.namespace_id AS resource_id
  FROM namespaces n WHERE n.namespace_id = $NamespaceId AND n.status = 'active'
  UNION ALL
  SELECT 'component' AS resource_kind, c.id AS resource_id
  FROM components c JOIN namespaces n ON n.namespace_id = c.namespace_id
  WHERE n.namespace_id = $NamespaceId AND n.status = 'active'
) s
LEFT JOIN resource_policies p ON p.tenant_id = $Tenant
 AND p.resource_kind = s.resource_kind AND p.resource_id = s.resource_id
LEFT JOIN resource_policy_revisions v
  ON v.tenant_id = p.tenant_id AND v.resource_kind = p.resource_kind
 AND v.resource_id = p.resource_id AND v.resource_version = p.resource_version
 AND v.revision = p.revision
ORDER BY s.resource_kind, s.resource_id, p.resource_version
