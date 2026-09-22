SELECT report."id", report."slug", report."title", report."description", report."owner_id", report."status", report."default_connector_name", report."component_scope", report."component_name", report."current_draft_version", report."etag", report."created_at", report."updated_at", report."deleted_at", report."namespace", report."should_delete" FROM  (
    SELECT r.*, '' AS should_delete
FROM reports r

)  report WHERE 1 = 1