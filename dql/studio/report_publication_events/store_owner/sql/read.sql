SELECT r.id, r.owner_id
FROM components r
WHERE r.id = $ReportId AND r.deleted_at IS NULL
  AND ($NamespaceId = '' OR EXISTS (SELECT 1 FROM namespaces n
       WHERE n.owner_id=r.owner_id AND n.name=r.namespace
         AND n.namespace_id=$NamespaceId AND n.deleted_at IS NULL AND n.status='active'))
UNION ALL
SELECT d.component_id AS id, d.owner_id
FROM component_deletions d
WHERE d.component_id = $ReportId
  AND ($NamespaceId = '' OR d.namespace_id=$NamespaceId)
  AND NOT EXISTS (SELECT 1 FROM components r WHERE r.id = d.component_id)
