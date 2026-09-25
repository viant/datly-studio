SELECT report."id", report."current_draft_version", report."etag", report."updated_at" FROM  (SELECT r.id, r.current_draft_version, r.etag, r.updated_at
FROM reports r
WHERE r.deleted_at IS NULL
)  report WHERE 1 = 1