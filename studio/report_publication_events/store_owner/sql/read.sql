SELECT report."id", report."owner_id" FROM  (SELECT r.id, r.owner_id
FROM reports r
WHERE r.id = $ReportId AND r.deleted_at IS NULL
)  report WHERE 1 = 1