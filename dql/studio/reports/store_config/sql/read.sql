SELECT r.id, r.namespace, r.slug, r.title, r.description, r.owner_id,
       r.status, r.default_connector_name, r.component_scope, r.component_name,
       r.current_draft_version, r.etag, r.updated_at, r.deleted_at
FROM reports r
