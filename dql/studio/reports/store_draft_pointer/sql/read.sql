SELECT r.id, r.current_draft_version, r.etag, r.updated_at
FROM reports r
WHERE r.deleted_at IS NULL
