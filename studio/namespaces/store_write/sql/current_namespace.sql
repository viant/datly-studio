SELECT r."owner_id", r."name", r."title", r."description", r."status", r."etag", r."created_at", r."updated_at", r."deleted_at" FROM (SELECT namespace."owner_id", namespace."name", namespace."title", namespace."description", namespace."status", namespace."etag", namespace."created_at", namespace."updated_at", namespace."deleted_at" FROM  (SELECT n.owner_id, n.name, n.title, n.description, n.status, n.etag,
       n.created_at, n.updated_at, n.deleted_at
FROM namespaces n
WHERE n.deleted_at IS NULL
)  namespace WHERE 1 = 1) r WHERE $criteria.CompositeIn("r", $NamespaceKeys)