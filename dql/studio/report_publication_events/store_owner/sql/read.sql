SELECT r.id, r.owner_id
FROM components r
WHERE r.id = $ReportId AND r.deleted_at IS NULL
