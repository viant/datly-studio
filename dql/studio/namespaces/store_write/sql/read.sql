SELECT n.owner_id, n.name, n.title, n.description, n.status, n.etag,
       n.created_at, n.updated_at, n.deleted_at
FROM namespaces n
WHERE n.deleted_at IS NULL
