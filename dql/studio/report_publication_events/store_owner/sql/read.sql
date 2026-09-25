SELECT r.id, r.owner_id
FROM reports r
WHERE r.id = $ReportId AND r.deleted_at IS NULL
