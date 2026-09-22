SELECT r."owner_id", r."name", r."title", r."status", r."etag", r."description", r."created_at", r."updated_at", r."deleted_at" FROM (SELECT namespace."owner_id", namespace."name", namespace."title", namespace."description", namespace."status", namespace."etag", namespace."created_at", namespace."updated_at", namespace."deleted_at", namespace."should_delete" FROM  (
    SELECT n.*, '' AS should_delete
FROM namespaces n
WHERE n.owner_id = $Auth.Auth.Subject

)  namespace WHERE 1 = 1) r WHERE $criteria.CompositeIn("r", $NamespaceKeys)