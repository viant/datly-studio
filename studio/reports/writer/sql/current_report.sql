SELECT r."namespace", r."id", r."slug", r."title", r."owner_id", r."status", r."default_connector_name", r."component_scope", r."component_name", r."etag", r."description", r."current_draft_version", r."created_at", r."updated_at", r."deleted_at" FROM (SELECT report."id", report."slug", report."title", report."description", report."owner_id", report."status", report."default_connector_name", report."component_scope", report."component_name", report."current_draft_version", report."etag", report."created_at", report."updated_at", report."deleted_at", report."namespace", report."should_delete" FROM  (
    SELECT r.*, '' AS should_delete
FROM reports r

)  report WHERE 1 = 1) r WHERE $criteria.CompositeIn("r", $ReportKeys)