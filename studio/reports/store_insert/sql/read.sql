SELECT report."id", report."namespace", report."slug", report."title", report."description", report."owner_id", report."status", report."default_connector_name", report."component_scope", report."component_name", report."etag", report."created_at", report."updated_at" FROM  (SELECT r.id, r.namespace, r.slug, r.title, r.description, r.owner_id,
       r.status, r.default_connector_name, r.component_scope, r.component_name,
       r.etag, r.created_at, r.updated_at
FROM reports r
)  report