SELECT COUNT(1) AS used FROM reports
WHERE owner_id = $OwnerId AND namespace = $Name AND deleted_at IS NULL
