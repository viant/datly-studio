SELECT * FROM  (SELECT n.owner_id,n.name,n.etag,n.updated_at FROM namespaces n WHERE n.deleted_at IS NULL
)  namespace