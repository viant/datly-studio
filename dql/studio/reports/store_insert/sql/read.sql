SELECT r.id, r.namespace, r.slug, r.title, r.description, r.owner_id,
       r.status, r.default_connector_name, r.component_scope, r.component_name,
       r.etag, r.created_at, r.updated_at
FROM reports r
